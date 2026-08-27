// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

func TestFormatTaskDuration_Seconds(t *testing.T) {
	result := formatTaskDuration(30 * time.Second)
	require.Equal(t, "30 seconds ago", result)
}

func TestFormatTaskDuration_Minutes(t *testing.T) {
	result := formatTaskDuration(5 * time.Minute)
	require.Equal(t, "5 minutes ago", result)
}

func TestFormatTaskDuration_Hours(t *testing.T) {
	result := formatTaskDuration(3 * time.Hour)
	require.Equal(t, "3 hours ago", result)
}

func TestFormatTaskDuration_Days(t *testing.T) {
	result := formatTaskDuration(3 * 24 * time.Hour)
	require.Equal(t, "3 days ago", result)
}

func TestFormatTaskDuration_Weeks(t *testing.T) {
	result := formatTaskDuration(14 * 24 * time.Hour)
	require.Equal(t, "2 weeks ago", result)
}

func TestFormatTaskDuration_Months(t *testing.T) {
	result := formatTaskDuration(60 * 24 * time.Hour)
	require.Equal(t, "2 months ago", result)
}

func TestSortTasksByServiceAndTime_ByName(t *testing.T) {
	now := time.Now()
	tasks := []TaskEntry{
		{ServiceName: "web", CreatedAt: now},
		{ServiceName: "api", CreatedAt: now},
	}
	sortTasksByServiceAndTime(tasks)
	require.Equal(t, "api", tasks[0].ServiceName)
	require.Equal(t, "web", tasks[1].ServiceName)
}

func TestSortTasksByServiceAndTime_ByTime(t *testing.T) {
	now := time.Now()
	tasks := []TaskEntry{
		{ServiceName: "web", CreatedAt: now.Add(-time.Hour)},
		{ServiceName: "web", CreatedAt: now},
	}
	sortTasksByServiceAndTime(tasks)
	// newest first within same service
	require.True(t, tasks[0].CreatedAt.After(tasks[1].CreatedAt))
}

func TestSortTasksByServiceAndTime_Empty(t *testing.T) {
	var tasks []TaskEntry
	sortTasksByServiceAndTime(tasks) // should not panic
	require.Empty(t, tasks)
}

func TestTaskEntry_StatusText(t *testing.T) {
	tests := []struct {
		name string
		e    TaskEntry
		want string
	}{
		{
			name: "no pull in flight falls back to the swarm task state",
			e:    TaskEntry{CurrentState: "running 3 minutes ago"},
			want: "running 3 minutes ago",
		},
		{
			name: "a pull in flight replaces the bare preparing state",
			e:    TaskEntry{CurrentState: "preparing 2 minutes ago", PullProgress: "pulling · 3/12 layers · 412 MB"},
			want: "pulling · 3/12 layers · 412 MB",
		},
		{
			name: "empty entry yields empty status",
			e:    TaskEntry{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.e.StatusText())
		})
	}
}

// Both loaders must carry the raw task state, not just the display string it is
// folded into: it is the only thing telling a replica swarm stopped on purpose
// apart from one that died, and the views tint on it (issue #601).
func TestGetTasksForServiceAndStack_CarryTheRawState(t *testing.T) {
	t.Cleanup(InvalidateSnapshot)
	SetSnapshot(&SwarmSnapshot{
		Services: []swarm.Service{{
			ID: "svc1",
			Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{
				Name:   "mystack_web",
				Labels: map[string]string{"com.docker.stack.namespace": "mystack"},
			}},
		}},
		Tasks: []swarm.Task{{
			ID: "task1", ServiceID: "svc1", Slot: 1,
			DesiredState: swarm.TaskStateShutdown,
			Status: swarm.TaskStatus{
				State:     swarm.TaskStateShutdown,
				Timestamp: time.Now().Add(-11 * time.Minute),
			},
		}},
	})

	forService, err := GetTasksForService("svc1")
	require.NoError(t, err)
	require.Len(t, forService, 1)
	require.Equal(t, "shutdown", forService[0].State)
	require.Equal(t, "shutdown 11 minutes ago", forService[0].CurrentState)

	forStack, err := GetTasksForStack("mystack")
	require.NoError(t, err)
	require.Len(t, forStack, 1)
	require.Equal(t, "shutdown", forStack[0].State)
}
