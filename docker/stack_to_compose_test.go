// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestFilterLabels_RemovesDockerStackKeys(t *testing.T) {
	input := map[string]string{
		"com.docker.stack.namespace": "mystack",
		"com.docker.stack.image":     "nginx:latest",
		"app":                        "web",
	}
	got := filterLabels(input)
	require.Equal(t, map[string]string{"app": "web"}, got)
}

func TestFilterLabels_KeepsUserLabels(t *testing.T) {
	input := map[string]string{
		"app":     "web",
		"version": "1.0",
	}
	got := filterLabels(input)
	require.Equal(t, map[string]string{"app": "web", "version": "1.0"}, got)
}

func TestFilterLabels_ReturnsNilWhenAllFiltered(t *testing.T) {
	input := map[string]string{
		"com.docker.stack.namespace": "mystack",
	}
	require.Nil(t, filterLabels(input))
}

func TestFilterLabels_ReturnsNilForEmptyMap(t *testing.T) {
	require.Nil(t, filterLabels(map[string]string{}))
}

func TestFilterLabels_NilInput(t *testing.T) {
	require.Nil(t, filterLabels(nil))
}

func TestComposeService_DeployLabelsYAML(t *testing.T) {
	cs := ComposeService{
		Image: "nginx:latest",
		Deploy: map[string]any{
			"replicas": 3,
			"labels": map[string]string{
				"com.example.role": "frontend",
			},
		},
	}

	out, err := yaml.Marshal(cs)
	require.NoError(t, err)
	yamlStr := string(out)

	require.Contains(t, yamlStr, "deploy:")
	require.Contains(t, yamlStr, "labels:")
	require.Contains(t, yamlStr, "com.example.role: frontend")
}

func TestComposeService_ServiceLevelLabelsYAML(t *testing.T) {
	cs := ComposeService{
		Image: "nginx:latest",
		Labels: map[string]string{
			"container.label": "value",
		},
	}

	out, err := yaml.Marshal(cs)
	require.NoError(t, err)
	yamlStr := string(out)

	require.Contains(t, yamlStr, "labels:")
	require.Contains(t, yamlStr, "container.label: value")
	require.NotContains(t, yamlStr, "deploy:")
}

func TestComposeService_BothLabelLevelsYAML(t *testing.T) {
	cs := ComposeService{
		Image: "nginx:latest",
		Labels: map[string]string{
			"container.label": "cval",
		},
		Deploy: map[string]any{
			"replicas": 2,
			"labels": map[string]string{
				"deploy.label": "dval",
			},
		},
	}

	out, err := yaml.Marshal(cs)
	require.NoError(t, err)

	// Unmarshal back to verify structure
	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(out, &parsed))

	// Service-level labels
	svcLabels, ok := parsed["labels"].(map[string]any)
	require.True(t, ok, "service-level labels should exist")
	require.Equal(t, "cval", svcLabels["container.label"])

	// Deploy labels
	deploy, ok := parsed["deploy"].(map[string]any)
	require.True(t, ok, "deploy section should exist")
	deployLabels, ok := deploy["labels"].(map[string]any)
	require.True(t, ok, "deploy.labels should exist")
	require.Equal(t, "dval", deployLabels["deploy.label"])
}
