package docker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types"
)

// InspectType enumerates supported resource types for inspect
type InspectType string

const (
	InspectNode      InspectType = "node"
	InspectService   InspectType = "service"
	InspectContainer InspectType = "container"
	InspectStack     InspectType = "stack"
)

// Inspect fetches and returns formatted JSON for the given Docker object.
func Inspect(ctx context.Context, t InspectType, id string) (string, error) {
	cli, err := GetClient()
	if err != nil {
		return "", fmt.Errorf("docker client: %w", err)
	}

	var obj any

	switch t {
	case InspectNode:
		node, _, err := cli.NodeInspectWithRaw(ctx, id)
		if err != nil {
			return "", fmt.Errorf("node inspect: %w", err)
		}
		obj = node

	case InspectService:
		svc, _, err := cli.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
		if err != nil {
			return "", fmt.Errorf("service inspect: %w", err)
		}
		obj = svc

	case InspectContainer:
		ctr, err := cli.ContainerInspect(ctx, id)
		if err != nil {
			return "", fmt.Errorf("container inspect: %w", err)
		}
		obj = ctr

	default:
		return "", fmt.Errorf("unsupported inspect type: %s", t)
	}

	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal inspect result: %w", err)
	}
	return string(pretty), nil
}
