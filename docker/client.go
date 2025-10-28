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

// GetClient returns a singleton Docker client.
func GetClient() (*client.Client, error) {
	dockerClientOnce.Do(func() {
		var err error
		dockerClient, err = client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			dockerClientErr = err
			return
		}
		dockerClient.NegotiateAPIVersion(context.Background())
	})
	return dockerClient, dockerClientErr
}

// Ping checks connection to the Docker daemon.
func Ping() error {
	c, err := GetClient()
	if err != nil {
		return err
	}
	_, err = c.Ping(context.Background())
	return err
}
