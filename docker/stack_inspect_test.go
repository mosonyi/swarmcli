// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// Issue #430: the inspect path must filter Docker-internal labels and surface
// container-level labels, mirroring the edit path. GetStackInspection itself
// needs a live Docker snapshot, so these unit tests lock the ServiceSummary
// contract those two changes rely on (the wiring is exercised end-to-end by
// integration-tests/docker/stack_inspect_test.go).

func TestServiceSummary_LabelsFilteredAndContainerLabelsSurfaced(t *testing.T) {
	// Mirrors how GetStackInspection populates the summary: filterLabels on the
	// service-level labels, container labels under a separate key.
	summary := ServiceSummary{
		Name:            "web",
		Labels:          filterLabels(map[string]string{"com.docker.stack.namespace": "demo", "tier": "frontend"}),
		ContainerLabels: filterLabels(map[string]string{"com.docker.stack.image": "nginx", "role": "proxy"}),
	}
	require.Equal(t, map[string]string{"tier": "frontend"}, summary.Labels)
	require.Equal(t, map[string]string{"role": "proxy"}, summary.ContainerLabels)

	out, err := json.Marshal(summary)
	require.NoError(t, err)
	js := string(out)
	require.NotContains(t, js, "com.docker.stack.", "internal stack labels must not leak into inspect JSON")
	require.Contains(t, js, `"container_labels"`)
	require.Contains(t, js, `"tier":"frontend"`)
	require.Contains(t, js, `"role":"proxy"`)
}

func TestServiceSummary_EmptyLabelMapsOmitted(t *testing.T) {
	// filterLabels returns nil when nothing user-facing remains; the omitempty
	// tags then drop both keys entirely.
	summary := ServiceSummary{
		Name:            "web",
		Labels:          filterLabels(map[string]string{"com.docker.stack.namespace": "demo"}),
		ContainerLabels: filterLabels(nil),
	}
	require.Nil(t, summary.Labels)
	require.Nil(t, summary.ContainerLabels)

	out, err := json.Marshal(summary)
	require.NoError(t, err)
	js := string(out)
	require.NotContains(t, js, "labels")
	require.NotContains(t, js, "container_labels")
}
