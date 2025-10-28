package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	swarmTypes "github.com/docker/docker/api/types/swarm"
)

func ListConfigs() ([]swarmTypes.Config, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}

	return c.ConfigList(context.Background(), types.ConfigListOptions{})
}

func InspectConfig(configID string) (*swarmTypes.Config, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}

	cfg, _, err := c.ConfigInspectWithRaw(context.Background(), configID)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func RemoveConfig(configID string) error {
	c, err := GetClient()
	if err != nil {
		return err
	}
	return c.ConfigRemove(context.Background(), configID)
}
