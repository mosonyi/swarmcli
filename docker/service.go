package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
)

type StackService struct {
	NodeID         string
	StackName      string
	ServiceName    string
	ServiceID      string
	ReplicasOnNode int
	ReplicasTotal  int
}

// ScaleService updates the replica count of a service by ID.
func ScaleService(serviceID string, replicas uint64) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer c.Close()

	ctx := context.Background()

	svc, _, err := c.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect service %s: %w", serviceID, err)
	}

	// Only replicated services can be scaled
	if svc.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not in replicated mode", svc.Spec.Name)
	}

	current := *svc.Spec.Mode.Replicated.Replicas
	if current == replicas {
		// Nothing to do
		return nil
	}

	svc.Spec.Mode.Replicated.Replicas = &replicas

	// Apply the update
	resp, err := c.ServiceUpdate(ctx, svc.ID, svc.Version, svc.Spec, types.ServiceUpdateOptions{})
	if err != nil {
		return fmt.Errorf("updating service %s replicas from %d to %d: %w", svc.Spec.Name, current, replicas, err)
	}

	if len(resp.Warnings) > 0 {
		for _, w := range resp.Warnings {
			fmt.Printf("⚠️  Warning scaling service %s: %s\n", svc.Spec.Name, w)
		}
	}

	return nil
}

// ScaleServiceByName looks up a service by name and scales it.
func ScaleServiceByName(serviceName string, replicas uint64) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer c.Close()

	ctx := context.Background()

	services, err := c.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return fmt.Errorf("listing services: %w", err)
	}

	var svcID string
	for _, svc := range services {
		if svc.Spec.Name == serviceName {
			svcID = svc.ID
			break
		}
	}

	if svcID == "" {
		return fmt.Errorf("service %s not found", serviceName)
	}

	return ScaleService(svcID, replicas)
}
