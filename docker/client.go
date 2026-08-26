// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"encoding/json"
	"fmt"
	swarmlog "github.com/Eldara-Tech/swarmcli/v2/utils/log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
)

var (
	cachedClient *client.Client
	// contextClients caches one client per explicitly named context, kept apart
	// from cachedClient so the ambient fast path stays exactly as it was: a warm
	// GetClient must not start resolving a context name on every call.
	contextClients map[string]*client.Client
	clientMu       sync.Mutex
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

// ClientFor returns a client for an explicitly named Docker context, cached per
// name.
//
// It is the seam that lets a caller address a specific swarm. GetClient builds
// from the session pin and caches one client for the whole process — correct
// for a single-swarm CLI, and unusable for anything that has to reconcile two
// swarms at once, because there is no argument by which to ask for the other
// one.
func ClientFor(ctxName string) (*client.Client, error) {
	if ctxName == "" {
		return nil, fmt.Errorf("docker context name is required")
	}
	clientMu.Lock()
	defer clientMu.Unlock()

	if cli, ok := contextClients[ctxName]; ok {
		return cli, nil
	}
	cli, err := buildClientFor(ctxName)
	if err != nil {
		return nil, err
	}
	if contextClients == nil {
		contextClients = make(map[string]*client.Client, 1)
	}
	contextClients[ctxName] = cli
	return cli, nil
}

// ResetClient closes every cached client and clears the caches so the next
// GetClient or ClientFor call creates a fresh connection. Safe to call when
// nothing has been cached yet. Named clients are dropped too: a context switch
// is not the only reason to reset, and a `docker context update` invalidates a
// pinned client just as surely as the ambient one.
func ResetClient() {
	clientMu.Lock()
	defer clientMu.Unlock()

	if cachedClient != nil {
		_ = cachedClient.Close()
		cachedClient = nil
	}
	for name, cli := range contextClients {
		_ = cli.Close()
		delete(contextClients, name)
	}
}

// buildClient performs the actual work of resolving the Docker context,
// creating an SDK client, and pinging the daemon.
func buildClient() (*client.Client, error) {
	ctxName, err := GetContextFromEnv()
	if err != nil {
		return nil, err
	}
	return buildClientFor(ctxName)
}

// buildClientFor creates and pings a client for one named context.
func buildClientFor(ctxName string) (*client.Client, error) {
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

	opts, err := clientOptsFor(host, ca, cert, key, skipVerify)
	if err != nil {
		return nil, err
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

// clientOptsFor builds the SDK options for one resolved endpoint.
//
// A scheme the SDK cannot dial itself — ssh:// today — is tunnelled through a
// connection helper. client.WithHost alone would hand the URL to
// go-connections, which special-cases only unix and npipe and TCP-dials
// everything else, so an ssh context failed at the ping.
func clientOptsFor(host, ca, cert, key string, skipVerify bool) ([]client.Opt, error) {
	helper, err := connhelper.GetConnectionHelper(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve a connection helper for '%s': %w", host, err)
	}

	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if helper != nil {
		// The helper owns the transport: helper.Host is a dummy URL the HTTP
		// requests are addressed to, and helper.Dialer runs
		// `ssh … docker system dial-stdio`. TLS is deliberately not applied —
		// the connection is secured by ssh, and any certs in the context's
		// storage describe a tcp:// endpoint this context is not using.
		l().Infof("[GetClient] using a connection helper for %q", host)
		return append(opts, client.WithHost(helper.Host), client.WithDialContext(helper.Dialer)), nil
	}

	opts = append(opts, client.WithHost(host))
	// Configure TLS if certs exist
	if fileExists(ca) && fileExists(cert) && fileExists(key) {
		opts = append(opts, client.WithTLSClientConfig(ca, cert, key))
	} else if skipVerify {
		l().Infof("[GetClient] skipVerify=true but no certs found")
	}
	return opts, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetCurrentContext returns the name of the Docker context this session uses.
//
// It reports the session pin, not whatever ~/.docker/config.json says at the
// moment of the call: the name shown in the header has to be the name the
// client is connected to. See SessionContext.
func GetCurrentContext() (string, error) {
	return SessionContext()
}

// GetContextFromEnv returns the docker context to use: the DOCKER_CONTEXT
// environment variable if set (so the app can be run against a specific
// context, e.g. in CI or local testing), otherwise the context that was active
// when the session started. The returned string will not contain a trailing
// newline.
//
// The resolution happens once — see SessionContext for why this does not
// follow a `docker context use` run in another terminal.
func GetContextFromEnv() (string, error) {
	return SessionContext()
}
