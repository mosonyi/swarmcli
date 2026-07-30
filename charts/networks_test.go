// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExternalNetworks(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		want     []string
	}{
		{
			name:     "external true uses the map key",
			manifest: "networks:\n  traefik-public:\n    external: true\n",
			want:     []string{"traefik-public"},
		},
		{
			name:     "external long form uses the resolved name",
			manifest: "networks:\n  alias:\n    external:\n      name: actual-net\n",
			want:     []string{"actual-net"},
		},
		{
			name:     "non-external networks are ignored",
			manifest: "networks:\n  internal:\n    driver: overlay\n  pub:\n    external: true\n",
			want:     []string{"pub"},
		},
		{
			name:     "external false is ignored",
			manifest: "networks:\n  x:\n    external: false\n",
			want:     nil,
		},
		{
			name:     "no networks block",
			manifest: "version: \"3.9\"\nservices:\n  app:\n    image: x\n",
			want:     nil,
		},
		{
			name:     "sorted and de-keyed across multiple",
			manifest: "networks:\n  b-net:\n    external: true\n  a-net:\n    external: true\n",
			want:     []string{"a-net", "b-net"},
		},
		{
			name:     "sibling name wins over the map key",
			manifest: "networks:\n  alias:\n    external: true\n    name: traefik-public\n",
			want:     []string{"traefik-public"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := externalNetworks(tc.manifest)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestExternalNetworksRejectsBothForms(t *testing.T) {
	_, err := externalNetworks("networks:\n  alias:\n    external:\n      name: deprecated\n    name: current\n")
	require.EqualError(t, err, `network "alias": external.name and name conflict; use only name`)
}
