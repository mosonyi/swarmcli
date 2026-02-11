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
		Nodes: []swarm.Node{{}, {}, {}},
	}
	stack, desired := getServiceStackAndDesired(svc, snap)
	require.Equal(t, "mystack", stack)
	require.Equal(t, 3, desired)
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

func TestCountTasksForNode_All(t *testing.T) {
	snap := &SwarmSnapshot{
		Tasks: []swarm.Task{
			{ServiceID: "svc1", NodeID: "n1", DesiredState: swarm.TaskStateRunning},
			{ServiceID: "svc1", NodeID: "n2", DesiredState: swarm.TaskStateRunning},
			{ServiceID: "svc2", NodeID: "n1", DesiredState: swarm.TaskStateRunning},
		},
	}
	require.Equal(t, 2, countTasksForNode("svc1", "", snap))
}

func TestCountTasksForNode_Specific(t *testing.T) {
	snap := &SwarmSnapshot{
		Tasks: []swarm.Task{
			{ServiceID: "svc1", NodeID: "n1", DesiredState: swarm.TaskStateRunning},
			{ServiceID: "svc1", NodeID: "n2", DesiredState: swarm.TaskStateRunning},
		},
	}
	require.Equal(t, 1, countTasksForNode("svc1", "n1", snap))
}

func TestCountTasksForNode_SkipsNonRunning(t *testing.T) {
	snap := &SwarmSnapshot{
		Tasks: []swarm.Task{
			{ServiceID: "svc1", NodeID: "n1", DesiredState: swarm.TaskStateRunning},
			{ServiceID: "svc1", NodeID: "n1", DesiredState: swarm.TaskStateShutdown},
		},
	}
	require.Equal(t, 1, countTasksForNode("svc1", "", snap))
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

func TestCreateSecretRevealService(t *testing.T) {
	spec := CreateSecretRevealService("reveal-svc", "secret-id-123", "my-secret")
	require.Equal(t, "reveal-svc", spec.Annotations.Name)
	require.Equal(t, "true", spec.Annotations.Labels["swarmcli.temporary"])
	require.Equal(t, "reveal-secret", spec.Annotations.Labels["swarmcli.purpose"])
	require.Equal(t, "alpine:latest", spec.TaskTemplate.ContainerSpec.Image)
	require.Len(t, spec.TaskTemplate.ContainerSpec.Secrets, 1)
	require.Equal(t, "secret-id-123", spec.TaskTemplate.ContainerSpec.Secrets[0].SecretID)
	require.Equal(t, "my-secret", spec.TaskTemplate.ContainerSpec.Secrets[0].SecretName)
	require.NotNil(t, spec.Mode.Replicated)
	require.Equal(t, uint64(1), *spec.Mode.Replicated.Replicas)
}

func TestCreateSecretRevealServiceWithImage_Override(t *testing.T) {
	spec := CreateSecretRevealServiceWithImage("svc", "busybox:latest", "sid", "sname")
	require.Equal(t, "busybox:latest", spec.TaskTemplate.ContainerSpec.Image)
}

func TestCreateSecretRevealServiceWithImage_EmptyFallback(t *testing.T) {
	spec := CreateSecretRevealServiceWithImage("svc", "", "sid", "sname")
	require.Equal(t, "alpine:latest", spec.TaskTemplate.ContainerSpec.Image)
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
