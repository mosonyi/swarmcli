package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	swarmlog "swarmcli/utils/log"

	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

func l() *zap.SugaredLogger {
	return swarmlog.Logger.With("docker", "client")
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
func GetClient() (*client.Client, error) {
	ctxNameBytes, err := exec.Command("docker", "context", "show").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get docker context: %w", err)
	}
	ctxName := string(ctxNameBytes)
	if len(ctxName) > 0 && ctxName[len(ctxName)-1] == '\n' {
		ctxName = ctxName[:len(ctxName)-1]
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

	l().Infoln("[GetClient] host=%q tlsPath=%q skipVerify=%v", host, tlsPath, skipVerify)
	l().Infoln("[GetClient] certs present: ca=%t cert=%t key=%t",
		fileExists(ca), fileExists(cert), fileExists(key))

	opts := []client.Opt{
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	}

	// Configure TLS if certs exist
	if fileExists(ca) && fileExists(cert) && fileExists(key) {
		opts = append(opts, client.WithTLSClientConfig(ca, cert, key))
	} else if skipVerify {
		l().Infoln("[GetClient] skipVerify=true but no certs found")
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// Verify connection
	if _, err := cli.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	return cli, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
