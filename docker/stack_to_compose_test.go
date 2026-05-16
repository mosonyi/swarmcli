// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"strings"
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

// Issue #363 regression: a stack declaring ONLY the "default" network with
// a service referencing it must survive reconstruction unchanged. The old
// suppression logic (removed) dropped both, causing "Edit Stack no network".
func TestPruneEmptySections_KeepsSoleDefaultNetwork(t *testing.T) {
	cf := ComposeFile{
		Version: "3.9",
		Services: map[string]ComposeService{
			"web": {Image: "nginx", Networks: []string{"default"}},
		},
		Networks: map[string]map[string]any{"default": {"driver": "overlay"}},
	}

	pruneEmptySections(&cf)

	require.NotNil(t, cf.Networks)
	require.Contains(t, cf.Networks, "default")
	require.Equal(t, "overlay", cf.Networks["default"]["driver"])

	out, err := yaml.Marshal(&cf)
	require.NoError(t, err)
	yamlStr := string(out)
	require.Contains(t, yamlStr, "networks:")
	require.Contains(t, yamlStr, "default:")
	require.Contains(t, yamlStr, "driver: overlay")
}

func TestPruneEmptySections_NilsGenuinelyEmpty(t *testing.T) {
	cf := ComposeFile{
		Version:  "3.9",
		Services: map[string]ComposeService{"web": {Image: "nginx"}},
		Networks: map[string]map[string]any{},
		Volumes:  map[string]map[string]any{},
		Secrets:  map[string]map[string]any{},
		Configs:  map[string]map[string]any{},
	}

	pruneEmptySections(&cf)

	require.Nil(t, cf.Networks)
	require.Nil(t, cf.Volumes)
	require.Nil(t, cf.Secrets)
	require.Nil(t, cf.Configs)
}

func TestStripImageDigest(t *testing.T) {
	digest := strings.Repeat("a", 64)
	tests := []struct{ name, in, want string }{
		{"digest pinned", "nginx:1.25@sha256:" + digest, "nginx:1.25"},
		{"no tag with digest", "nginx@sha256:" + digest, "nginx"},
		{"registry port + digest", "reg:5000/app:v2@sha256:" + digest, "reg:5000/app:v2"},
		{"plain tag", "alpine:latest", "alpine:latest"},
		{"no tag no digest", "alpine", "alpine"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, stripImageDigest(tt.in))
		})
	}
}

func TestFormatPort(t *testing.T) {
	tests := []struct {
		name              string
		published, target uint32
		proto, want       string
	}{
		{"tcp default omitted", 8080, 80, "tcp", "8080:80"}, // issue #363 case
		{"empty proto = tcp", 8080, 80, "", "8080:80"},
		{"udp kept", 53, 53, "udp", "53:53/udp"},
		{"sctp kept", 38412, 38412, "sctp", "38412:38412/sctp"},
		{"target only tcp", 0, 80, "tcp", "80"},
		{"target only udp", 0, 514, "udp", "514/udp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, formatPort(tt.published, tt.target, tt.proto))
		})
	}
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

func TestEscapeComposeInterpolation(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"no dollars", "echo hello", "echo hello"},
		{"single dollar", "echo $HOME", "echo $$HOME"},
		{"double dollar passthrough", "echo $$HOME", "echo $$$$HOME"},
		{"subshell", "sh -c '$(date)'", "sh -c '$$(date)'"},
		{"arithmetic", "i=$((i+1))", "i=$$((i+1))"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, escapeComposeInterpolation(tt.input))
		})
	}
}

func TestEscapeComposeArgs(t *testing.T) {
	in := []string{"sh", "-c", "echo $HOME; date=$(date)"}
	got := escapeComposeArgs(in)
	require.Equal(t, []string{"sh", "-c", "echo $$HOME; date=$$(date)"}, got)
	// original slice must not be modified
	require.Equal(t, "echo $HOME; date=$(date)", in[2])
}
