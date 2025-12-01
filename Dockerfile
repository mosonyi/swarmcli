# syntax=docker/dockerfile:1
FROM golang:1.25

# Set working directory
WORKDIR /app

# Install Docker CLI
RUN apt-get update && \
    apt-get install -y \
        ca-certificates \
        curl \
        gnupg \
        lsb-release && \
    curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg && \
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list && \
    apt-get update && \
    apt-get install -y docker-ce-cli && \
    rm -rf /var/lib/apt/lists/*

# Copy go.mod and go.sum separately
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Default command (optional, override with `docker run`)
CMD ["bash"]
