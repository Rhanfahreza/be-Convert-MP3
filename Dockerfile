# Stage 1: Build the static Go binary
FROM golang:alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server main.go

# Stage 2: Minimal runtime image with ffmpeg and standalone yt-dlp
FROM alpine:3.20

RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    curl \
    python3 \
    bash

# Download official yt-dlp executable binary
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp \
    && chmod a+rx /usr/local/bin/yt-dlp

WORKDIR /app

COPY --from=builder /app/server /app/server

# Create directory for downloads
RUN mkdir -p /app/downloads

ENV PORT=8080
EXPOSE 8080

CMD ["/app/server"]
