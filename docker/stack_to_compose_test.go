// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

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

func TestComposeHealthcheck_FullSpec(t *testing.T) {
	hc := composeHealthcheck(
		[]string{"CMD", "curl", "-f", "http://localhost"},
		90*int64(time.Second), 10*int64(time.Second),
		40*int64(time.Second), 5*int64(time.Second), 3, false)
	require.NotNil(t, hc)
	require.Equal(t, []string{"CMD", "curl", "-f", "http://localhost"}, hc.Test)
	require.Equal(t, "1m30s", hc.Interval)
	require.Equal(t, "10s", hc.Timeout)
	require.Equal(t, "40s", hc.StartPeriod)
	require.Equal(t, "5s", hc.StartInterval)
	require.Equal(t, 3, hc.Retries)
	require.False(t, hc.Disable)
}

func TestComposeHealthcheck_Disabled(t *testing.T) {
	hc := composeHealthcheck([]string{"NONE"}, 0, 0, 0, 0, 0, false)
	require.NotNil(t, hc)
	require.True(t, hc.Disable)
	require.Empty(t, hc.Test)
	require.Empty(t, hc.Interval)
}

func TestComposeHealthcheck_InheritReturnsNil(t *testing.T) {
	require.Nil(t, composeHealthcheck(nil, 0, 0, 0, 0, 0, false))
	require.Nil(t, composeHealthcheck([]string{}, 0, 0, 0, 0, 0, false))
}

func TestComposeHealthcheck_Escapes(t *testing.T) {
	hc := composeHealthcheck(
		[]string{"CMD-SHELL", "test $VAR = 1"}, int64(time.Second), 0, 0, 0, 0, true)
	require.NotNil(t, hc)
	require.Equal(t, []string{"CMD-SHELL", "test $$VAR = 1"}, hc.Test)
}

func TestComposeService_HealthcheckYAML(t *testing.T) {
	cs := ComposeService{
		Image: "nginx:latest",
		Healthcheck: &Healthcheck{
			Test:     []string{"CMD", "true"},
			Interval: "30s",
			Timeout:  "5s",
			Retries:  3,
		},
	}
	out, err := yaml.Marshal(&cs)
	require.NoError(t, err)
	yamlStr := string(out)
	require.Contains(t, yamlStr, "healthcheck:")
	require.Contains(t, yamlStr, "test:")
	require.Contains(t, yamlStr, "interval: 30s")
	require.Contains(t, yamlStr, "retries: 3")
}

func TestComposeService_HealthcheckOmittedWhenNil(t *testing.T) {
	cs := ComposeService{Image: "nginx:latest"}
	out, err := yaml.Marshal(&cs)
	require.NoError(t, err)
	require.NotContains(t, string(out), "healthcheck")
}

// Issue #428: a service deployed with a logging driver+options must round-trip
// into the compose `logging:` block.
func TestComposeLogging_DriverAndOptions(t *testing.T) {
	lg := composeLogging("loki:latest", map[string]string{
		"loki-url": "http://loki:3100/loki/api/v1/push",
	})
	require.NotNil(t, lg)
	require.Equal(t, "loki:latest", lg.Driver)
	require.Equal(t, "http://loki:3100/loki/api/v1/push", lg.Options["loki-url"])
}

func TestComposeLogging_DriverOnly(t *testing.T) {
	lg := composeLogging("json-file", nil)
	require.NotNil(t, lg)
	require.Equal(t, "json-file", lg.Driver)
	require.Empty(t, lg.Options)
}

func TestComposeLogging_OptionsOnly(t *testing.T) {
	lg := composeLogging("", map[string]string{"max-size": "10m"})
	require.NotNil(t, lg)
	require.Empty(t, lg.Driver)
	require.Equal(t, "10m", lg.Options["max-size"])
}

func TestComposeLogging_NoDriverReturnsNil(t *testing.T) {
	require.Nil(t, composeLogging("", nil))
	require.Nil(t, composeLogging("", map[string]string{}))
}

func TestComposeService_LoggingYAML(t *testing.T) {
	cs := ComposeService{
		Image: "nginx:latest",
		Logging: &Logging{
			Driver:  "loki:latest",
			Options: map[string]string{"loki-url": "http://loki:3100/loki/api/v1/push"},
		},
	}
	out, err := yaml.Marshal(&cs)
	require.NoError(t, err)
	yamlStr := string(out)
	require.Contains(t, yamlStr, "logging:")
	require.Contains(t, yamlStr, "driver: loki:latest")
	require.Contains(t, yamlStr, "options:")
	require.Contains(t, yamlStr, "loki-url: http://loki:3100/loki/api/v1/push")
}

func TestComposeService_LoggingOmittedWhenNil(t *testing.T) {
	cs := ComposeService{Image: "nginx:latest"}
	out, err := yaml.Marshal(&cs)
	require.NoError(t, err)
	require.NotContains(t, string(out), "logging")
}

// Capture side: the `docker service inspect` JSON LogDriver block must
// unmarshal into TaskTemplate.LogDriver (issue #428).
func TestServiceInspect_CapturesLogDriver(t *testing.T) {
	raw := []byte(`{
		"Spec": {
			"Name": "web",
			"TaskTemplate": {
				"LogDriver": {
					"Name": "loki:latest",
					"Options": {"loki-url": "http://loki:3100/loki/api/v1/push"}
				}
			}
		}
	}`)
	var si ServiceInspect
	require.NoError(t, json.Unmarshal(raw, &si))
	require.NotNil(t, si.Spec.TaskTemplate.LogDriver)
	require.Equal(t, "loki:latest", si.Spec.TaskTemplate.LogDriver.Name)
	require.Equal(t, "http://loki:3100/loki/api/v1/push",
		si.Spec.TaskTemplate.LogDriver.Options["loki-url"])
}

func TestServiceInspect_NoLogDriverIsNil(t *testing.T) {
	raw := []byte(`{"Spec":{"Name":"web","TaskTemplate":{}}}`)
	var si ServiceInspect
	require.NoError(t, json.Unmarshal(raw, &si))
	require.Nil(t, si.Spec.TaskTemplate.LogDriver)
}
