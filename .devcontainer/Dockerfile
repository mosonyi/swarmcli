FROM mcr.microsoft.com/vscode/devcontainers/go:1.23

RUN apt-get update && apt-get install -y docker.io && rm -rf /var/lib/apt/lists/*

ARG USERNAME=vscode
RUN usermod -aG docker ${USERNAME}

ENV GONOSUMDB=* \
    GONOPROXY=* \
    GOPROXY=https://proxy.golang.org,direct