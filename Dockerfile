# syntax=docker/dockerfile:1
FROM golang:1.25

# Set working directory
WORKDIR /app

# Install Docker CLI and plugins for Debian Trixie
RUN apt-get update && \
    apt-get install -y \
        ca-certificates \
        curl \
        gnupg && \
    install -m 0755 -d /etc/apt/keyrings && \
    curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg && \
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian trixie stable" > /etc/apt/sources.list.d/docker.list && \
    apt-get update && \
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin && \
    rm -rf /var/lib/apt/lists/*

# Copy go.mod and go.sum separately
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Default command (optional, override with `docker run`)
CMD ["bash"]
