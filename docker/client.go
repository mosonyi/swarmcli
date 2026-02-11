// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	swarmlog "swarmcli/utils/log"
	"sync"
	"time"

	"github.com/docker/docker/client"
)

var (
	cachedClient *client.Client
	clientMu     sync.Mutex
)

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("docker", "client")
}

type dockerContext struct {
	Endpoints struct {
		Docker struct {
			Host          string `json:"Host"`
			SkipTLSVerify bool   `json:"SkipTLSVerify"`
		} `json:"docker"`
	} `json:"Endpoints"`
	Storage struct {
		TLSPath string `json:"TLSPath"`
	} `json:"Storage"`
}

// GetClient returns a Docker SDK client configured based on the current Docker context.
// The client is cached as a package-level singleton; subsequent calls return the
// cached instance without spawning subprocesses or pinging the daemon. Call
// ResetClient to force a fresh client (e.g. after a context switch).
func GetClient() (*client.Client, error) {
	clientMu.Lock()
	defer clientMu.Unlock()

	if cachedClient != nil {
		return cachedClient, nil
	}

	cli, err := buildClient()
	if err != nil {
		return nil, err
	}
	cachedClient = cli
	return cachedClient, nil
}

// ResetClient closes the cached client (if any) and clears the cache so the
// next GetClient call creates a fresh connection. Safe to call when no client
// has been cached yet.
func ResetClient() {
	clientMu.Lock()
	defer clientMu.Unlock()

	if cachedClient != nil {
		_ = cachedClient.Close()
		cachedClient = nil
	}
}

// buildClient performs the actual work of resolving the Docker context,
// creating an SDK client, and pinging the daemon.
func buildClient() (*client.Client, error) {
	ctxName, err := GetContextFromEnv()
	if err != nil {
		return nil, err
	}

	inspectOut, err := exec.Command("docker", "context", "inspect", ctxName).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect context: %w", err)
	}

	var contexts []dockerContext
	if err := json.Unmarshal(inspectOut, &contexts); err != nil {
		return nil, fmt.Errorf("failed to parse context JSON: %w", err)
	}
	if len(contexts) == 0 {
		return nil, fmt.Errorf("no context info found for %s", ctxName)
	}
	ctx := contexts[0]

	host := ctx.Endpoints.Docker.Host
	skipVerify := ctx.Endpoints.Docker.SkipTLSVerify
	tlsPath := ctx.Storage.TLSPath

	// If certs are in a subfolder named "docker", prefer that
	dockerTLSPath := filepath.Join(tlsPath, "docker")
	if stat, err := os.Stat(dockerTLSPath); err == nil && stat.IsDir() {
		tlsPath = dockerTLSPath
	}

	ca := filepath.Join(tlsPath, "ca.pem")
	cert := filepath.Join(tlsPath, "cert.pem")
	key := filepath.Join(tlsPath, "key.pem")

	l().Infof("[GetClient] host=%q tlsPath=%q skipVerify=%v", host, tlsPath, skipVerify)
	l().Infof("[GetClient] certs present: ca=%t cert=%t key=%t",
		fileExists(ca), fileExists(cert), fileExists(key))

	opts := []client.Opt{
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	}

	// Configure TLS if certs exist
	if fileExists(ca) && fileExists(cert) && fileExists(key) {
		opts = append(opts, client.WithTLSClientConfig(ca, cert, key))
	} else if skipVerify {
		l().Infof("[GetClient] skipVerify=true but no certs found")
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// Verify connection with timeout
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(pingCtx); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	return cli, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetCurrentContext returns the name of the active Docker context.
func GetCurrentContext() (string, error) {
	ctxNameBytes, err := exec.Command("docker", "context", "show").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get docker context: %w", err)
	}
	ctxName := string(ctxNameBytes)
	if len(ctxName) > 0 && ctxName[len(ctxName)-1] == '\n' {
		ctxName = ctxName[:len(ctxName)-1]
	}
	return ctxName, nil
}

// GetContextFromEnv returns the docker context to use. It prefers the
// DOCKER_CONTEXT environment variable (so the app can be run against a
// specific context, e.g. in CI or local testing). If that variable is not
// set, it falls back to calling `docker context show` to retrieve the active
// context. The returned string will not contain a trailing newline.
func GetContextFromEnv() (string, error) {
	ctxName := os.Getenv("DOCKER_CONTEXT")
	if ctxName != "" {
		return ctxName, nil
	}

	ctxNameBytes, err := exec.Command("docker", "context", "show").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get docker context: %w", err)
	}
	ctxName = string(ctxNameBytes)
	if len(ctxName) > 0 && ctxName[len(ctxName)-1] == '\n' {
		ctxName = ctxName[:len(ctxName)-1]
	}
	return ctxName, nil
}
