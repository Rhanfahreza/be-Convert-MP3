# Convert-MP3 (Go Fiber v3)

Aplikasi web berkecepatan tinggi berbasis **Go (Golang) Fiber v3** untuk mengekstrak & mengonversi audio dari TikTok & YouTube ke format MP3 berkualitas tinggi, kompatibel dengan smartwatch (**Huawei Watch Fit 4**).

## Tech Stack
- **Framework**: [Fiber v3](https://github.com/gofiber/fiber/tree/v3) (v3.6.1) berbasis Fasthttp
- **Language**: Go (Golang)
- **Engine Unduhan & Tagging**: `yt-dlp` + `ffmpeg`
- **UI**: Modern Minimalist Utility (Geist font + Tailwind CSS)
- **Auto Cleanup**: Background Goroutine membersihkan file unduhan sementara setiap 5 menit.

## Cara Menjalankan

Cukup jalankan script:
```bash
./start.sh
```
Atau jalankan langsung:
```bash
go run main.go
```

- Buka di laptop: [http://localhost:8000](http://localhost:8000)
- Buka di HP (1 Wi-Fi): `http://192.168.1.35:8000`
