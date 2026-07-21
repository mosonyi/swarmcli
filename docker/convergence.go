// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"time"

	"github.com/docker/docker/api/types/swarm"
)

// ServiceConvergence is what a caller needs to decide whether one service has
// finished rolling out. It is deliberately separate from ServiceEntry: that
// struct backs the services view, and its ReplicasOnNode counts tasks by
// DESIRED state, which answers "what does the orchestrator intend" rather than
// "what is actually up" (see issue #480).
type ServiceConvergence struct {
	Name string
	Mode string
	// Running counts tasks that are actually running, on an active node.
	Running int
	// Desired is the target count over active nodes.
	Desired int
	// UpdateState is the raw swarm UpdateStatus.State, empty when the service
	// has never been updated. Note that a nil UpdateStatus means "no rollout has
	// ever run", NOT "the rollout finished".
	UpdateState string
	// Monitor is UpdateConfig.Monitor: the window after a task is created during
	// which its failure still counts against the rollout. Zero when unset.
	Monitor time.Duration
}

// activeNodeIDs returns the nodes that can currently run tasks. A task pinned to
// a node that is down never converges, so counting it in the denominator would
// make a release hang until timeout for a reason no redeploy can fix.
func activeNodeIDs(snap *SwarmSnapshot) map[string]struct{} {
	active := make(map[string]struct{}, len(snap.Nodes))
	for _, n := range snap.Nodes {
		if n.Status.State == swarm.NodeStateReady && n.Spec.Availability == swarm.NodeAvailabilityActive {
			active[n.ID] = struct{}{}
		}
	}
	return active
}

// LoadStackConvergence returns per-service convergence facts for a stack.
//
// Running counts tasks whose ACTUAL state is running — not their desired state.
// Up-to-dateness comes free: on a rolling update Swarm marks superseded tasks
// DesiredState=shutdown, so requiring both DesiredState and Status.State to be
// running counts exactly the current generation.
func LoadStackConvergence(stackName string) []ServiceConvergence {
	snap, err := GetOrRefreshSnapshot()
	if err != nil {
		l().Warnf("failed to get snapshot: %v", err)
		return nil
	}
	active := activeNodeIDs(snap)

	var out []ServiceConvergence
	for _, svc := range snap.Services {
		if svc.Spec.Labels["com.docker.stack.namespace"] != stackName {
			continue
		}

		running := 0
		for _, t := range snap.Tasks {
			if t.ServiceID != svc.ID {
				continue
			}
			if t.DesiredState != swarm.TaskStateRunning || t.Status.State != swarm.TaskStateRunning {
				continue
			}
			// A task not yet assigned has no NodeID; it is by definition not
			// running yet, but guard anyway rather than counting it as active.
			if t.NodeID == "" {
				continue
			}
			if _, ok := active[t.NodeID]; !ok {
				continue
			}
			running++
		}

		out = append(out, ServiceConvergence{
			Name:        svc.Spec.Name,
			Mode:        getServiceMode(svc),
			Running:     running,
			Desired:     desiredOverActiveNodes(svc, len(active)),
			UpdateState: updateState(svc),
			Monitor:     monitorWindow(svc),
		})
	}
	return out
}

// desiredOverActiveNodes is the target task count. For a global service that is
// one per active node, so draining a node lowers the target rather than making
// the service permanently short.
func desiredOverActiveNodes(svc swarm.Service, activeNodes int) int {
	switch {
	case svc.Spec.Mode.Replicated != nil && svc.Spec.Mode.Replicated.Replicas != nil:
		return int(*svc.Spec.Mode.Replicated.Replicas)
	case svc.Spec.Mode.Global != nil:
		return activeNodes
	default:
		return 1
	}
}

func updateState(svc swarm.Service) string {
	if svc.UpdateStatus == nil {
		return ""
	}
	return string(svc.UpdateStatus.State)
}

func monitorWindow(svc swarm.Service) time.Duration {
	if svc.Spec.UpdateConfig == nil {
		return 0
	}
	return svc.Spec.UpdateConfig.Monitor
}
