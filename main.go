package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/google/uuid"
)

const (
	downloadsDir   = "downloads"
	maxFileSize    = "100M"            // Batasi download maksimal 100MB agar disk laptop tidak penuh
	processTimeout = 5 * time.Minute   // Timeout 5 menit untuk mencegah livestream/video 10 jam menggantung
)

// Semaphore untuk membatasi maksimal 2 proses konversi ffmpeg bersamaan
// agar CPU laptop tidak drop/overheat
var jobSemaphore = make(chan struct{}, 2)

// Daftar domain media yang diperbolehkan (Whitelisting untuk proteksi total dari SSRF)
var allowedDomains = []string{
	"youtube.com",
	"www.youtube.com",
	"m.youtube.com",
	"youtu.be",
	"music.youtube.com",
	"tiktok.com",
	"www.tiktok.com",
	"m.tiktok.com",
	"vm.tiktok.com",
	"vt.tiktok.com",
	"instagram.com",
	"www.instagram.com",
	"soundcloud.com",
	"m.soundcloud.com",
	"twitter.com",
	"x.com",
	"facebook.com",
	"fb.watch",
}

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

// Sanitasi dan validasi ketat URL untuk mencegah Argument Injection & SSRF
func validateAndSanitizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("URL tidak boleh kosong")
	}

	// Mencegah argument injection jika ada input yang dimulai dengan '-'
	if strings.HasPrefix(rawURL, "-") {
		return "", fmt.Errorf("Format URL tidak valid")
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", fmt.Errorf("URL tidak valid")
	}

	// Wajib hanya http atau https
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("Protokol hanya boleh HTTP atau HTTPS")
	}

	// Validasi hostname (Domain Allowlist)
	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return "", fmt.Errorf("Hostname URL tidak valid")
	}

	// Blokir akses ke IP private/internal (Anti-SSRF)
	if hostname == "localhost" || strings.HasPrefix(hostname, "127.") || hostname == "::1" ||
		strings.HasPrefix(hostname, "192.168.") || strings.HasPrefix(hostname, "10.") ||
		strings.HasPrefix(hostname, "172.") || strings.HasPrefix(hostname, "169.254.") {
		return "", fmt.Errorf("Akses ke jaringan lokal/internal dilarang")
	}

	// Pastikan domain termasuk dalam platform yang didukung
	isAllowed := false
	for _, domain := range allowedDomains {
		if hostname == domain || strings.HasSuffix(hostname, "."+domain) {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return "", fmt.Errorf("Domain %s tidak didukung. Harap gunakan tautan YouTube, TikTok, atau platform resmi lainnya.", hostname)
	}

	return parsed.String(), nil
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

func sanitizeErrorMessage(errStr string) string {
	// Sembunyikan path direktori internal sistem
	home, _ := os.UserHomeDir()
	if home != "" {
		errStr = strings.ReplaceAll(errStr, home, "~")
	}
	return errStr
}

func main() {
	_ = os.MkdirAll(downloadsDir, 0755)
	startBackgroundCleanup()

	app := fiber.New(fiber.Config{
		ServerHeader: "Fiber-v3",
		AppName:      "beConvert-MP3",
		BodyLimit:    64 * 1024, // Maksimal payload 64 KB (Mencegah serangan Memory Exhaustion)
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

	// Rate limiter untuk endpoint convert: Maksimal 10 request per menit per IP
	convertLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(ErrorResponse{
				Detail: "Terlalu banyak permintaan konversi. Tunggu 1 menit sebelum mencoba lagi.",
			})
		},
	})

	// API: Convert to MP3
	app.Post("/api/convert", convertLimiter, func(c fiber.Ctx) error {
		var req ConvertRequest
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Detail: "Format JSON tidak valid"})
		}

		// 1. Validasi & Sanitasi URL (Anti-SSRF & Anti Argument Injection)
		cleanURL, err := validateAndSanitizeURL(req.URL)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Detail: err.Error()})
		}

		// 2. Validasi Bitrate (Whitelisting ketat)
		bitrate := req.Bitrate
		if bitrate != "128" && bitrate != "192" && bitrate != "320" {
			bitrate = "192"
		}

		// 3. Batasi proses bersamaan (Anti-CPU Spikes / Anti-DoS)
		select {
		case jobSemaphore <- struct{}{}:
			defer func() { <-jobSemaphore }()
		default:
			return c.Status(fiber.StatusTooManyRequests).JSON(ErrorResponse{
				Detail: "Server sedang sibuk memproses antrean audio lain. Mohon tunggu sebentar lalu coba lagi.",
			})
		}

		taskID := uuid.New().String()
		taskDir := filepath.Join(downloadsDir, taskID)
		if err := os.MkdirAll(taskDir, 0755); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Detail: "Gagal mengalokasikan ruang kerja",
			})
		}

		outputTemplate := filepath.Join(taskDir, "%(title)s.%(ext)s")

		// 4. yt-dlp arguments dengan proteksi:
		// - Menggunakan POSIX '--' sebelum URL agar input URL tidak bisa disisipi command flags
		// - Batas ukuran maksimal file (--max-filesize)
		// - Batas timeout eksekusi (CommandContext)
		ctx, cancel := context.WithTimeout(context.Background(), processTimeout)
		defer cancel()

		args := []string{
			"--no-playlist",
			"-x",
			"--audio-format", "mp3",
			"--audio-quality", bitrate,
			"--add-metadata",
			"--convert-thumbnails", "jpg",
			"--max-filesize", maxFileSize,
			"-o", outputTemplate,
			"--retries", "3",
		}

		if req.EmbedThumbnail {
			args = append(args, "--embed-thumbnail")
		}

		// '--' memisahkan options dari positional arguments (URL) secara mutlak
		args = append(args, "--", cleanURL)

		ytDlpBinary := getYtDlpPath()
		cmd := exec.CommandContext(ctx, ytDlpBinary, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			_ = os.RemoveAll(taskDir)
			if ctx.Err() == context.DeadlineExceeded {
				return c.Status(fiber.StatusRequestTimeout).JSON(ErrorResponse{
					Detail: "Proses konversi melebihi batas waktu (maksimal 5 menit). Durasi media mungkin terlalu panjang.",
				})
			}

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
				Detail: sanitizeErrorMessage(fmt.Sprintf("Error konversi: %s", lastErr)),
			})
		}

		// Temukan file .mp3 yang dihasilkan
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

	// API: Download file (Dilindungi dari Path Traversal)
	app.Get("/api/download/:task_id", func(c fiber.Ctx) error {
		taskID := c.Params("task_id")

		// Validasi ketat format UUID (Mencegah Directory Traversal / Path Traversal)
		if _, err := uuid.Parse(taskID); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid Task ID")
		}

		taskDir := filepath.Join(downloadsDir, taskID)

		// Verifikasi bahwa taskDir benar-benar berada di dalam downloadsDir
		cleanBase, _ := filepath.Abs(downloadsDir)
		cleanTarget, _ := filepath.Abs(taskDir)
		if !strings.HasPrefix(cleanTarget, cleanBase) {
			return c.Status(fiber.StatusForbidden).SendString("Access Denied")
		}

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
	fmt.Printf(" beConvert-MP3 (Hardened & Protected) Running on port %s!\n", port)
	fmt.Printf(" Access locally  : http://localhost:%s\n", port)
	fmt.Printf(" Access from phone: http://%s:%s\n", ip, port)
	fmt.Println("==========================================================")

	if err := app.Listen(":" + port); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
