// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

func TestGetServiceStatus_Active(t *testing.T) {
	svc := swarm.Service{UpdateStatus: nil}
	require.Equal(t, "active", getServiceStatus(svc))
}

func TestGetServiceStatus_Updating(t *testing.T) {
	tests := []struct {
		state    swarm.UpdateState
		expected string
	}{
		{swarm.UpdateStateUpdating, "updating"},
		{swarm.UpdateStatePaused, "paused"},
		{swarm.UpdateStateCompleted, "updated"},
		{swarm.UpdateStateRollbackStarted, "rolling back"},
		{swarm.UpdateStateRollbackPaused, "rollback paused"},
		{swarm.UpdateStateRollbackCompleted, "rolled back"},
	}
	for _, tc := range tests {
		t.Run(string(tc.state), func(t *testing.T) {
			svc := swarm.Service{
				UpdateStatus: &swarm.UpdateStatus{State: tc.state},
			}
			require.Equal(t, tc.expected, getServiceStatus(svc))
		})
	}
}

func TestGetServiceStackAndDesired_Replicated(t *testing.T) {
	replicas := uint64(3)
	svc := swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{Labels: map[string]string{"com.docker.stack.namespace": "mystack"}},
			Mode:        swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
		},
	}
	snap := &SwarmSnapshot{}
	stack, desired := getServiceStackAndDesired(svc, snap)
	require.Equal(t, "mystack", stack)
	require.Equal(t, 3, desired)
}

func TestGetServiceStackAndDesired_Global(t *testing.T) {
	svc := swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{Labels: map[string]string{"com.docker.stack.namespace": "mystack"}},
			Mode:        swarm.ServiceMode{Global: &swarm.GlobalService{}},
		},
	}
	snap := &SwarmSnapshot{
		Nodes: []swarm.Node{activeNode("n1"), activeNode("n2"), activeNode("n3")},
	}
	stack, desired := getServiceStackAndDesired(svc, snap)
	require.Equal(t, "mystack", stack)
	require.Equal(t, 3, desired)
}

// A global service targets one task per node that can actually run one. Counting
// a drained or down node in the denominator made a fully healthy service read
// permanently short (issue #480).
func TestGetServiceStackAndDesired_GlobalIgnoresInactiveNodes(t *testing.T) {
	svc := swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{Labels: map[string]string{"com.docker.stack.namespace": "mystack"}},
			Mode:        swarm.ServiceMode{Global: &swarm.GlobalService{}},
		},
	}
	drained := activeNode("n3")
	drained.Spec.Availability = swarm.NodeAvailabilityDrain
	down := activeNode("n4")
	down.Status.State = swarm.NodeStateDown

	snap := &SwarmSnapshot{Nodes: []swarm.Node{activeNode("n1"), activeNode("n2"), drained, down}}
	_, desired := getServiceStackAndDesired(svc, snap)
	require.Equal(t, 2, desired)
}

func activeNode(id string) swarm.Node {
	return swarm.Node{
		ID:     id,
		Status: swarm.NodeStatus{State: swarm.NodeStateReady},
		Spec:   swarm.NodeSpec{Availability: swarm.NodeAvailabilityActive},
	}
}

func TestGetServiceStackAndDesired_NoStack(t *testing.T) {
	replicas := uint64(1)
	svc := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
		},
	}
	snap := &SwarmSnapshot{}
	stack, _ := getServiceStackAndDesired(svc, snap)
	require.Equal(t, "-", stack)
}

// runningTask is a task that is actually up: swarm intends to run it AND the
// container reports running.
func runningTask(svcID, nodeID string) swarm.Task {
	return swarm.Task{
		ServiceID:    svcID,
		NodeID:       nodeID,
		DesiredState: swarm.TaskStateRunning,
		Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
	}
}

func TestCountTasksForNode_All(t *testing.T) {
	snap := &SwarmSnapshot{
		Tasks: []swarm.Task{
			runningTask("svc1", "n1"),
			runningTask("svc1", "n2"),
			runningTask("svc2", "n1"),
		},
	}
	require.Equal(t, 2, countTasksForNode("svc1", "", snap))
}

func TestCountTasksForNode_Specific(t *testing.T) {
	snap := &SwarmSnapshot{
		Tasks: []swarm.Task{runningTask("svc1", "n1"), runningTask("svc1", "n2")},
	}
	require.Equal(t, 1, countTasksForNode("svc1", "n1", snap))
}

func TestCountTasksForNode_SkipsNonRunning(t *testing.T) {
	shutdown := runningTask("svc1", "n1")
	shutdown.DesiredState = swarm.TaskStateShutdown
	shutdown.Status.State = swarm.TaskStateShutdown

	snap := &SwarmSnapshot{Tasks: []swarm.Task{runningTask("svc1", "n1"), shutdown}}
	require.Equal(t, 1, countTasksForNode("svc1", "", snap))
}

// The bug this fixes: a task swarm INTENDS to run but which has not started —
// pending on an unsatisfiable placement constraint, or still pulling an image —
// was counted as if it were up, so the column read 3/3 where `docker service ls`
// read 1/3 and the service never converged (issue #480).
func TestCountTasksForNode_SkipsScheduledButNotYetRunning(t *testing.T) {
	scheduled := func(state swarm.TaskState) swarm.Task {
		t := runningTask("svc1", "n1")
		t.Status.State = state // still intended to run, not there yet
		return t
	}
	snap := &SwarmSnapshot{
		Tasks: []swarm.Task{
			runningTask("svc1", "n1"),
			scheduled(swarm.TaskStatePending),
			scheduled(swarm.TaskStateStarting),
			scheduled(swarm.TaskStatePreparing),
		},
	}
	require.Equal(t, 1, countTasksForNode("svc1", "", snap))
}

// A superseded task is still counted while it runs, matching `docker service
// ls`. Up-to-dateness is a separate question, and LoadStackConvergence is where
// --wait asks it.
func TestCountTasksForNode_CountsSupersededTasksStillRunning(t *testing.T) {
	outgoing := runningTask("svc1", "n1")
	outgoing.DesiredState = swarm.TaskStateShutdown // superseded, container still up

	snap := &SwarmSnapshot{Tasks: []swarm.Task{runningTask("svc1", "n2"), outgoing}}
	require.Equal(t, 2, countTasksForNode("svc1", "", snap))
}

func TestSortEntries(t *testing.T) {
	entries := []ServiceEntry{
		{StackName: "beta", ServiceName: "web"},
		{StackName: "alpha", ServiceName: "api"},
		{StackName: "alpha", ServiceName: "db"},
	}
	sortEntries(entries)
	require.Equal(t, "alpha", entries[0].StackName)
	require.Equal(t, "api", entries[0].ServiceName)
	require.Equal(t, "alpha", entries[1].StackName)
	require.Equal(t, "db", entries[1].ServiceName)
	require.Equal(t, "beta", entries[2].StackName)
}

func TestStripDockerLogHeaders_Empty(t *testing.T) {
	require.Equal(t, "", stripDockerLogHeaders(nil))
}

func TestStripDockerLogHeaders_Short(t *testing.T) {
	require.Equal(t, "hello", stripDockerLogHeaders([]byte("hello")))
}

func TestStripDockerLogHeaders_NotMultiplexed(t *testing.T) {
	data := []byte("plain text log line that is longer than 8 bytes")
	require.Equal(t, string(data), stripDockerLogHeaders(data))
}

func TestStripDockerLogHeaders_SingleFrame(t *testing.T) {
	payload := []byte("hello world")
	header := make([]byte, 8)
	header[0] = 1 // stdout
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	frame := append(header, payload...)
	require.Equal(t, "hello world", stripDockerLogHeaders(frame))
}

func TestStripDockerLogHeaders_MultipleFrames(t *testing.T) {
	var frames []byte
	for _, msg := range []string{"hello ", "world"} {
		header := make([]byte, 8)
		header[0] = 1
		binary.BigEndian.PutUint32(header[4:], uint32(len(msg)))
		frames = append(frames, header...)
		frames = append(frames, []byte(msg)...)
	}
	require.Equal(t, "hello world", stripDockerLogHeaders(frames))
}

func TestStripDockerLogHeaders_InvalidSize(t *testing.T) {
	header := make([]byte, 8)
	header[0] = 1
	binary.BigEndian.PutUint32(header[4:], 9999) // size exceeds available data
	frame := append(header, []byte("short")...)
	result := stripDockerLogHeaders(frame)
	require.Equal(t, string(frame), result, "should fallback to raw on invalid frame")
}

func TestTrySendProgress_BufferedChannel(t *testing.T) {
	ch := make(chan ProgressUpdate, 1)
	trySendProgress(ch, ProgressUpdate{Replaced: 1, Running: 2, Total: 3})
	got := <-ch
	require.Equal(t, 1, got.Replaced)
	require.Equal(t, 2, got.Running)
	require.Equal(t, 3, got.Total)
}

func TestTrySendProgress_FullChannel(t *testing.T) {
	ch := make(chan ProgressUpdate) // unbuffered
	done := make(chan struct{})
	go func() {
		trySendProgress(ch, ProgressUpdate{Replaced: 1, Total: 1})
		close(done)
	}()
	select {
	case <-done:
		// good - did not block
	case <-time.After(time.Second):
		t.Fatal("trySendProgress blocked on full channel")
	}
}
