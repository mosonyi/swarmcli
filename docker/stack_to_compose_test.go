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

func TestComposeFile_VolumesExternalTrue(t *testing.T) {
	cf := ComposeFile{
		Version:  "3.8",
		Services: map[string]ComposeService{"web": {Image: "nginx"}},
		Volumes: map[string]map[string]any{
			"mydata": {"external": true},
		},
	}
	out, err := yaml.Marshal(&cf)
	require.NoError(t, err)
	yamlStr := string(out)
	require.Contains(t, yamlStr, "external: true")
}

func TestComposeFile_VolumeWithDriver(t *testing.T) {
	cf := ComposeFile{
		Version:  "3.8",
		Services: map[string]ComposeService{"web": {Image: "nginx", Volumes: []string{"nfs_data:/data"}}},
		Volumes: map[string]map[string]any{
			"nfs_data": {
				"driver":      "local",
				"driver_opts": map[string]string{"type": "nfs", "device": ":/export"},
			},
		},
	}
	out, err := yaml.Marshal(&cf)
	require.NoError(t, err)
	yamlStr := string(out)
	require.Contains(t, yamlStr, "driver: local")
	require.Contains(t, yamlStr, "driver_opts:")
	require.NotContains(t, yamlStr, "external")
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

func TestStripStackPrefix(t *testing.T) {
	tests := []struct {
		stack, input, want string
	}{
		{"ubu", "ubu_ubuntu_example_data", "ubuntu_example_data"},
		{"ubu", "external_volume", "external_volume"},
		{"ubu", "ubu_default", "default"},
		{"ubu", "ubu_", ""},
		{"", "anything", "anything"},
		{"ubu", "ubu", "ubu"}, // no underscore → no strip
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.want, stripStackPrefix(tt.stack, tt.input))
		})
	}
}

func TestComposeFile_VolumeManagedNotExternal(t *testing.T) {
	cf := ComposeFile{
		Version:  "3.8",
		Services: map[string]ComposeService{"web": {Image: "nginx", Volumes: []string{"mydata:/data"}}},
		Volumes:  map[string]map[string]any{"mydata": {}},
	}
	out, err := yaml.Marshal(&cf)
	require.NoError(t, err)
	require.NotContains(t, string(out), "external")
}

func TestComposeFile_DefaultNetworkOmitted(t *testing.T) {
	cf := ComposeFile{
		Version: "3.8",
		Services: map[string]ComposeService{
			"web": {Image: "nginx", Networks: []string{"default"}},
		},
		Networks: map[string]map[string]any{"default": {"external": true}},
	}

	// Simulate the suppression logic
	if len(cf.Networks) == 1 {
		if _, ok := cf.Networks["default"]; ok {
			cf.Networks = nil
			for k, svc := range cf.Services {
				if nets, ok := svc.Networks.([]string); ok && len(nets) == 1 && nets[0] == "default" {
					svc.Networks = nil
					cf.Services[k] = svc
				}
			}
		}
	}

	out, err := yaml.Marshal(&cf)
	require.NoError(t, err)
	yamlStr := string(out)
	require.NotContains(t, yamlStr, "networks:")
	require.NotContains(t, yamlStr, "default")
}

func TestComposeFile_NetworkManagedNotExternal(t *testing.T) {
	// Stack-managed networks (prefix was stripped) should not be external
	cf := ComposeFile{
		Version: "3.8",
		Services: map[string]ComposeService{
			"web": {Image: "nginx", Networks: []string{"mynet"}},
		},
		Networks: map[string]map[string]any{"mynet": {}},
	}
	out, err := yaml.Marshal(&cf)
	require.NoError(t, err)
	yamlStr := string(out)
	require.Contains(t, yamlStr, "networks:")
	require.Contains(t, yamlStr, "mynet:")
	require.NotContains(t, yamlStr, "external")
}

func TestComposeFile_NetworkExternalTrue(t *testing.T) {
	// External networks (no prefix stripped) should have external: true
	cf := ComposeFile{
		Version: "3.8",
		Services: map[string]ComposeService{
			"web": {Image: "nginx", Networks: []string{"shared_net"}},
		},
		Networks: map[string]map[string]any{"shared_net": {"external": true}},
	}
	out, err := yaml.Marshal(&cf)
	require.NoError(t, err)
	yamlStr := string(out)
	require.Contains(t, yamlStr, "shared_net:")
	require.Contains(t, yamlStr, "external: true")
}
