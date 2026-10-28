#!/bin/bash
BACKEND_DIR="/home/c0mra1dddd/Documents/beConvert-MP3"

echo "=========================================================="
echo "🚀 Starting MP3 Converter Backend & Cloudflare Tunnel"
echo "Public API: https://api-mp3.ackteams.com"
echo "Tekan CTRL + C untuk mematikan backend & tunnel"
echo "=========================================================="

# Matikan proses lama jika ada di port 8000
fuser -k 8000/tcp 2>/dev/null

# Jalankan Go Backend dari folder backend yang tepat
cd "$BACKEND_DIR"
./server &
SERVER_PID=$!

# Handler saat tombol CTRL+C ditekan
cleanup() {
    echo ""
    echo "🛑 Mematikan server dan tunnel..."
    kill $SERVER_PID 2>/dev/null
    kill $TUNNEL_PID 2>/dev/null
    exit 0
}

trap cleanup INT TERM

# Tunggu sejenak agar backend siap di port 8000
sleep 1

# Jalankan Cloudflare Tunnel
cloudflared tunnel run convert-mp3 &
TUNNEL_PID=$!

wait
