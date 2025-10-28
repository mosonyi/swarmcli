package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
)

type StackService struct {
	NodeID      string
	StackName   string
	ServiceName string
}

func GetServiceNameToIDMap() (map[string]string, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}

	services, err := c.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	m := make(map[string]string, len(services))
	for _, s := range services {
		m[s.Spec.Name] = s.ID
	}
	return m, nil
}

func inspectStackServices(serviceIDs []string) ([]StackService, error) {
	if len(serviceIDs) == 0 {
		return nil, nil
	}

	c, err := GetClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	stackServices := make([]StackService, 0, len(serviceIDs))
	unique := make(map[string]struct{})

	for _, id := range serviceIDs {
		svc, _, err := c.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
		if err != nil {
			continue
		}
		stack := svc.Spec.Labels["com.docker.stack.namespace"]
		key := stack + "|" + svc.Spec.Name
		if _, exists := unique[key]; !exists {
			unique[key] = struct{}{}
			stackServices = append(stackServices, StackService{
				StackName:   stack,
				ServiceName: svc.Spec.Name,
			})
		}
	}
	return stackServices, nil
}
