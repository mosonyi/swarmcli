package docker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/docker/client"
)

// SwarmNode defined earlier in your code; repeated here if needed.
// type SwarmNode struct { ... }

var (
	cli  *client.Client
	once sync.Once
)

// dockerContextJSON is a minimal struct matching 'docker context inspect' output.
type dockerContextJSON struct {
	Name      string `json:"Name"`
	Endpoints map[string]struct {
		Host          string `json:"Host"`
		SkipTLSVerify bool   `json:"SkipTLSVerify"`
	} `json:"Endpoints"`
	TLSMaterial map[string][]string `json:"TLSMaterial"`
	Storage     struct {
		MetadataPath string `json:"MetadataPath"`
		TLSPath      string `json:"TLSPath"`
	} `json:"Storage"`
}

// GetClient creates a singleton docker client that uses the current CLI context.
// It will load TLS certs from Storage.TLSPath when present and configure a proper HTTPS http.Client.
func GetClient() (*client.Client, error) {
	var err error
	once.Do(func() {
		host, tlsPath, skipVerify, jerr := inspectDockerContext()
		if jerr != nil {
			err = jerr
			return
		}

		// debug prints — make sure your logger writes to a file when using bubbletea
		log.Printf("[GetClient] host=%q tlsPath=%q skipVerify=%v\n", host, tlsPath, skipVerify)

		opts := []client.Opt{client.WithHost(host)}
		var httpClient *http.Client

		// If we have a TLS path (directory with cert.pem, key.pem, ca.pem), build TLS http.Client
		// --- inside GetClient(), after getting tlsPath ---
		if tlsPath != "" {
			// check if there's a "docker" subdir that contains the certs
			dockerSubdir := filepath.Join(tlsPath, "docker")
			if fileExists(filepath.Join(dockerSubdir, "cert.pem")) {
				tlsPath = dockerSubdir
			}

			certFile := filepath.Join(tlsPath, "cert.pem")
			keyFile := filepath.Join(tlsPath, "key.pem")
			caFile := filepath.Join(tlsPath, "ca.pem")

			if fileExists(certFile) && fileExists(keyFile) && fileExists(caFile) {
				cert, loadErr := tls.LoadX509KeyPair(certFile, keyFile)
				if loadErr != nil {
					err = fmt.Errorf("failed to load client cert/key: %w", loadErr)
					return
				}

				caPEM, readErr := os.ReadFile(caFile)
				if readErr != nil {
					err = fmt.Errorf("failed to read ca.pem: %w", readErr)
					return
				}

				caPool := x509.NewCertPool()
				if !caPool.AppendCertsFromPEM(caPEM) {
					err = fmt.Errorf("failed to append CA certs from %s", caFile)
					return
				}

				tlsCfg := &tls.Config{
					Certificates: []tls.Certificate{cert},
					RootCAs:      caPool,
				}
				if skipVerify {
					tlsCfg.InsecureSkipVerify = true
				}

				httpClient = &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: tlsCfg,
					},
				}

				if strings.HasPrefix(host, "tcp://") {
					host = "https://" + strings.TrimPrefix(host, "tcp://")
				}
				opts = []client.Opt{client.WithHost(host), client.WithHTTPClient(httpClient)}
			} else {
				log.Printf("[GetClient] certs not found in %s or %s\n", tlsPath, dockerSubdir)
			}
		}

		if httpClient != nil {
			opts = append(opts, client.WithHTTPClient(httpClient))
		}

		cli, err = client.NewClientWithOpts(opts...)
		if err == nil {
			cli.NegotiateAPIVersion(context.Background())
		}
	})

	if err != nil {
		return nil, err
	}
	if cli == nil {
		return nil, fmt.Errorf("docker client is nil")
	}
	return cli, nil
}

// inspectDockerContext runs `docker context inspect` and returns host, TLS path (if any), and SkipTLSVerify.
func inspectDockerContext() (host string, tlsPath string, skipTLSVerify bool, err error) {
	out, err := exec.Command("docker", "context", "inspect").Output()
	if err != nil {
		return "", "", false, fmt.Errorf("docker context inspect failed: %w", err)
	}

	// The output is an array; unmarshall into slice.
	var ctxs []dockerContextJSON
	if err = json.Unmarshal(out, &ctxs); err != nil {
		return "", "", false, fmt.Errorf("failed to parse docker context inspect output: %w", err)
	}
	if len(ctxs) == 0 {
		return "", "", false, fmt.Errorf("no contexts returned by docker context inspect")
	}

	ctx := ctxs[0]

	ep, ok := ctx.Endpoints["docker"]
	if !ok {
		// sometimes the endpoint key might be lowercase or different; attempt first endpoint
		for _, e := range ctx.Endpoints {
			ep = e
			ok = true
			break
		}
		if !ok {
			return "", "", false, fmt.Errorf("no docker endpoint in context")
		}
	}

	host = strings.TrimSpace(ep.Host)
	skipTLSVerify = ep.SkipTLSVerify

	// Primary: prefer Storage.TLSPath if present (this contains actual cert directory)
	if ctx.Storage.TLSPath != "" {
		tlsPath = strings.TrimSpace(ctx.Storage.TLSPath)
		// make absolute if relative
		if !filepath.IsAbs(tlsPath) {
			if abs, aerr := filepath.Abs(tlsPath); aerr == nil {
				tlsPath = abs
			}
		}
		return host, tlsPath, skipTLSVerify, nil
	}

	// Secondary: older structures used TLSMaterial map to list filenames; if present, attempt to find them in common TLS directories
	if files, ok := ctx.TLSMaterial["docker"]; ok && len(files) > 0 {
		// Common place: ~/.docker/contexts/tls/<hash> is often where CLI stores them.
		home := os.Getenv("HOME")
		if home != "" {
			// guess TLSPath by scanning ~/.docker/contexts/tls/*
			tlsBase := filepath.Join(home, ".docker", "contexts", "tls")
			entries, derr := ioutil.ReadDir(tlsBase)
			if derr == nil {
				for _, ent := range entries {
					if !ent.IsDir() {
						continue
					}
					candidate := filepath.Join(tlsBase, ent.Name())
					// quick check if candidate contains the listed files
					allExist := true
					for _, fname := range files {
						if !fileExists(filepath.Join(candidate, fname)) {
							allExist = false
							break
						}
					}
					if allExist {
						return host, candidate, skipTLSVerify, nil
					}
				}
			}
		}
	}

	// No TLS path discovered
	return host, "", skipTLSVerify, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
