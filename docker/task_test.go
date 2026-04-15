// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"testing"
	"time"

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
