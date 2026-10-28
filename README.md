# Convert-MP3 (Go Fiber v3 + React TypeScript)

Aplikasi web modern untuk mengonversi audio dari TikTok & YouTube ke format MP3 berkualitas tinggi, siap diputar secara offline di smartwatch (**Huawei Watch Fit 4**).

## Tech Stack
- **Frontend**: [React 18](https://react.dev/) + [TypeScript](https://www.typescriptlang.org/) + [Vite](https://vitejs.dev/) + [Tailwind CSS](https://tailwindcss.com/) + [Lucide Icons](https://lucide.dev/)
- **Backend**: [Go Fiber v3](https://github.com/gofiber/fiber/tree/v3) (v3.6.1)
- **Audio Engine**: `yt-dlp` (standalone) + `ffmpeg`
- **Networking**: Deteksi otomatis IP Wi-Fi untuk kemudahan akses dari browser smartphone.

## Struktur Direktori
```text
Convert-MP3/
├── downloads/            # Folder unduhan sementara (auto-clean tiap 5 menit)
├── frontend/             # Frontend React 18 + Vite + TypeScript
│   ├── src/
│   │   ├── App.tsx       # Komponen utama React
│   │   ├── main.tsx
│   │   └── index.css
│   ├── dist/             # Hasil build produksi React
│   ├── package.json
│   └── vite.config.ts
├── go.mod                # Dependensi Go Fiber v3
├── go.sum
├── main.go               # Backend server Go Fiber v3
├── server                # Binary executable mandiri Go
├── start.sh              # Script run otomatis
└── README.md
```

## Cara Menjalankan

Jalankan langsung script utama:
```bash
./start.sh
```

Untuk mode development frontend (Vite HMR):
```bash
cd frontend && bun run dev
```

- Buka di laptop: [http://localhost:8000](http://localhost:8000)
- Buka di HP (1 Wi-Fi): `http://192.168.1.35:8000`
