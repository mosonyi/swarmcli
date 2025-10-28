package docker

import (
	"context"
	"sync"

	"github.com/docker/docker/client"
)

var (
	dockerClient     *client.Client
	dockerClientOnce sync.Once
	dockerClientErr  error
)

// GetClient returns a singleton Docker client that respects the environment's current Docker context.
func GetClient() (*client.Client, error) {
	dockerClientOnce.Do(func() {
		var err error
		// This automatically uses DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH etc.
		dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			dockerClientErr = err
			return
		}
	})
	return dockerClient, dockerClientErr
}

// Ping checks if the Docker daemon is reachable.
func Ping() error {
	c, err := GetClient()
	if err != nil {
		return err
	}
	_, err = c.Ping(context.Background())
	return err
}
