# syntax=docker/dockerfile:1
FROM golang:1.25

# Set working directory
WORKDIR /app

# Install Docker CLI from Debian repos to avoid docker.com 404s
RUN apt-get update && \
    apt-get install -y \
        ca-certificates \
        curl \
        gnupg \
        docker.io \
        docker-buildx-plugin \
        docker-compose-plugin && \
    rm -rf /var/lib/apt/lists/*

# Copy go.mod and go.sum separately
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Default command (optional, override with `docker run`)
CMD ["bash"]
