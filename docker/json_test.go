// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"encoding/json"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

func TestConfigJSON_KeyValueData(t *testing.T) {
	cfg := &ConfigWithDecodedData{
		Config: swarm.Config{Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: "test"}}},
		Data:   []byte("KEY=value\nFOO=bar"),
	}
	raw, err := cfg.JSON()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(raw, &parsed))
	dataParsed, ok := parsed["DataParsed"].(map[string]any)
	require.True(t, ok, "DataParsed should be a map for key=value data")
	require.Equal(t, "value", dataParsed["KEY"])
	require.Equal(t, "bar", dataParsed["FOO"])
}

func TestConfigJSON_RawData(t *testing.T) {
	cfg := &ConfigWithDecodedData{
		Config: swarm.Config{},
		Data:   []byte("this is not key=value format\njust plain text"),
	}
	raw, err := cfg.JSON()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(raw, &parsed))
	_, isString := parsed["DataParsed"].(string)
	require.True(t, isString, "DataParsed should be a string for non key=value data")
}

func TestConfigPrettyJSON(t *testing.T) {
	cfg := &ConfigWithDecodedData{
		Config: swarm.Config{},
		Data:   []byte("KEY=value"),
	}
	raw, err := cfg.PrettyJSON()
	require.NoError(t, err)
	require.Contains(t, string(raw), "\n")
	require.Contains(t, string(raw), "  ")
}

func TestSecretJSON(t *testing.T) {
	sec := &SecretWithDecodedData{
		Secret: swarm.Secret{Spec: swarm.SecretSpec{Annotations: swarm.Annotations{Name: "my-secret"}}},
	}
	raw, err := sec.JSON()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(raw, &parsed))
	require.Contains(t, parsed["DataParsed"], "write-only")
}

func TestSecretPrettyJSON(t *testing.T) {
	sec := &SecretWithDecodedData{
		Secret: swarm.Secret{},
	}
	raw, err := sec.PrettyJSON()
	require.NoError(t, err)
	require.Contains(t, string(raw), "\n")
	require.Contains(t, string(raw), "  ")
}

func TestNetworkJSON(t *testing.T) {
	nw := &NetworkWithUsage{
		Network:  network.Summary{Name: "my-net"},
		Services: []string{"web", "api"},
	}
	raw, err := nw.JSON()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(raw, &parsed))
	require.NotNil(t, parsed["Network"])
	services, ok := parsed["Services"].([]any)
	require.True(t, ok)
	require.Len(t, services, 2)
}
