#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd "$DIR"

# Build Frontend (React + TypeScript) if not built yet
if [ ! -d "frontend/dist" ]; then
    echo "Building React TypeScript Frontend..."
    cd frontend && bun install && bun run build && cd ..
fi

# Build Go Fiber v3 binary if needed
if [ ! -f "server" ]; then
    echo "Building Go Fiber v3 backend..."
    go build -o server main.go
fi

exec ./server
