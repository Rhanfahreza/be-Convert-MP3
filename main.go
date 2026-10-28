package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/google/uuid"
)

const downloadsDir = "downloads"

type ConvertRequest struct {
	URL            string `json:"url"`
	Bitrate        string `json:"bitrate"`
	EmbedThumbnail bool   `json:"embed_thumbnail"`
}

type ConvertResponse struct {
	TaskID      string `json:"task_id"`
	Filename    string `json:"filename"`
	Filesize    string `json:"filesize"`
	DownloadURL string `json:"download_url"`
}

type ErrorResponse struct {
	Detail string `json:"detail"`
}

type NetworkResponse struct {
	LanIP string `json:"lan_ip"`
	Port  int    `json:"port"`
}

func getYtDlpPath() string {
	home, err := os.UserHomeDir()
	if err == nil {
		localBin := filepath.Join(home, ".local", "bin", "yt-dlp")
		if _, err := os.Stat(localBin); err == nil {
			return localBin
		}
	}
	if p, err := exec.LookPath("yt-dlp"); err == nil {
		return p
	}
	return "yt-dlp"
}

func getLocalIP() string {
	// 1. Try checking Wi-Fi interface via ip command
	out, err := exec.Command("ip", "-4", "-o", "addr", "show", "scope", "global").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				iface := parts[1]
				ipCidr := parts[3]
				ip := strings.Split(ipCidr, "/")[0]
				if strings.HasPrefix(iface, "wl") || strings.Contains(iface, "wifi") {
					return ip
				}
			}
		}
	}

	// 2. Fallback: UDP dial trick to get outgoing IP
	conn, err := net.Dial("udp", "1.1.1.1:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		return localAddr.IP.String()
	}

	return "127.0.0.1"
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func startBackgroundCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			entries, err := os.ReadDir(downloadsDir)
			if err != nil {
				continue
			}

			now := time.Now()
			for _, entry := range entries {
				if entry.IsDir() {
					info, err := entry.Info()
					if err == nil {
						if now.Sub(info.ModTime()) > 15*time.Minute {
							_ = os.RemoveAll(filepath.Join(downloadsDir, entry.Name()))
						}
					}
				}
			}
		}
	}()
}

func main() {
	_ = os.MkdirAll(downloadsDir, 0755)
	startBackgroundCleanup()

	app := fiber.New(fiber.Config{
		ServerHeader: "Fiber-v3",
		AppName:      "beConvert-MP3",
	})

	origins := os.Getenv("ALLOWED_ORIGINS")
	allowOriginsList := []string{"*"}
	if origins != "" {
		allowOriginsList = strings.Split(origins, ",")
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins: allowOriginsList,
		AllowHeaders: []string{"Origin, Content-Type, Accept, Authorization"},
		AllowMethods: []string{"GET, POST, HEAD, OPTIONS"},
	}))

	// Root API Info
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "beConvert-MP3",
			"status":  "running",
			"lan_ip":  getLocalIP(),
			"endpoints": []string{
				"/api/network",
				"/api/convert",
				"/api/download/:task_id",
			},
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	// API: Get Network / LAN Info
	app.Get("/api/network", func(c fiber.Ctx) error {
		return c.JSON(NetworkResponse{
			LanIP: getLocalIP(),
			Port:  8000,
		})
	})

	// API: Convert to MP3
	app.Post("/api/convert", func(c fiber.Ctx) error {
		var req ConvertRequest
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Detail: "Invalid JSON request"})
		}

		req.URL = strings.TrimSpace(req.URL)
		if req.URL == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Detail: "URL tidak boleh kosong"})
		}

		bitrate := req.Bitrate
		if bitrate != "128" && bitrate != "192" && bitrate != "320" {
			bitrate = "192"
		}

		taskID := uuid.New().String()
		taskDir := filepath.Join(downloadsDir, taskID)
		if err := os.MkdirAll(taskDir, 0755); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Detail: fmt.Sprintf("Gagal membuat folder: %v", err),
			})
		}

		outputTemplate := filepath.Join(taskDir, "%(title)s.%(ext)s")

		args := []string{
			"--no-playlist",
			"-x",
			"--audio-format", "mp3",
			"--audio-quality", bitrate,
			"--add-metadata",
			"--convert-thumbnails", "jpg",
			"-o", outputTemplate,
			"--retries", "3",
		}

		if req.EmbedThumbnail {
			args = append(args, "--embed-thumbnail")
		}

		args = append(args, req.URL)

		ytDlpBinary := getYtDlpPath()
		cmd := exec.Command(ytDlpBinary, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			_ = os.RemoveAll(taskDir)
			outStr := strings.TrimSpace(string(out))
			lastErr := err.Error()
			if outStr != "" {
				lines := strings.Split(outStr, "\n")
				for i := len(lines) - 1; i >= 0; i-- {
					l := strings.TrimSpace(lines[i])
					if l != "" {
						lastErr = l
						break
					}
				}
			}
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Detail: fmt.Sprintf("yt-dlp error: %s", lastErr),
			})
		}

		// Locate the generated .mp3 file
		files, err := os.ReadDir(taskDir)
		if err != nil || len(files) == 0 {
			_ = os.RemoveAll(taskDir)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Detail: "File MP3 tidak ditemukan setelah proses selesai",
			})
		}

		var mp3File os.DirEntry
		for _, f := range files {
			if strings.EqualFold(filepath.Ext(f.Name()), ".mp3") {
				mp3File = f
				break
			}
		}

		if mp3File == nil {
			_ = os.RemoveAll(taskDir)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Detail: "File MP3 tidak ditemukan",
			})
		}

		info, _ := mp3File.Info()
		fileSize := info.Size()

		return c.JSON(ConvertResponse{
			TaskID:      taskID,
			Filename:    mp3File.Name(),
			Filesize:    formatBytes(fileSize),
			DownloadURL: fmt.Sprintf("/api/download/%s", taskID),
		})
	})

	// API: Download file
	app.Get("/api/download/:task_id", func(c fiber.Ctx) error {
		taskID := c.Params("task_id")
		taskDir := filepath.Join(downloadsDir, taskID)

		files, err := os.ReadDir(taskDir)
		if err != nil || len(files) == 0 {
			return c.Status(fiber.StatusNotFound).SendString("File tidak ditemukan atau sudah kedaluwarsa")
		}

		var targetPath string
		var targetName string
		for _, f := range files {
			if strings.EqualFold(filepath.Ext(f.Name()), ".mp3") {
				targetPath = filepath.Join(taskDir, f.Name())
				targetName = f.Name()
				break
			}
		}

		if targetPath == "" {
			return c.Status(fiber.StatusNotFound).SendString("File MP3 tidak ditemukan")
		}

		return c.Download(targetPath, targetName)
	})

	ip := getLocalIP()
	fmt.Println("==========================================================")
	fmt.Printf(" beConvert-MP3 (Go Fiber v3 API) Running on port %s!\n", port)
	fmt.Printf(" Access locally  : http://localhost:%s\n", port)
	fmt.Printf(" Access from phone: http://%s:%s\n", ip, port)
	fmt.Println("==========================================================")

	if err := app.Listen(":" + port); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
