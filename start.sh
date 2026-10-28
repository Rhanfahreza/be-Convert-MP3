#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd "$DIR"

# Ensure binary exists
if [ ! -f "server" ]; then
    echo "Building Go Fiber v3 binary..."
    go build -o server main.go
fi

exec ./server
