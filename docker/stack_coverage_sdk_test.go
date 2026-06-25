// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build sdkcoverage

// Opt-in advisory: build/run with `-tags sdkcoverage`. It is intentionally
// kept OUT of the default CI matrix so a routine Docker SDK bump does not
// red-bar unrelated PRs. Run it when bumping github.com/docker/docker to
// surface swarm spec fields that were added upstream but are not yet mapped
// (or explicitly waived) by the service→compose reconstruction (see #430).
//
//	go test -tags sdkcoverage ./docker/...

package docker

import (
	"reflect"
	"testing"

	"github.com/docker/docker/api/types/swarm"
)

// TestSDKFieldCoverage fails when a field on one of the reconstructed swarm
// specs is neither mapped by the reconstruction nor explicitly listed as
// intentionally dropped. This turns a future silent parse-drop into a loud,
// reviewable test failure: the contributor bumping the SDK must decide to map
// the new field or waive it here.
func TestSDKFieldCoverage(t *testing.T) {
	cases := []struct {
		name string
		typ  reflect.Type
		// covered maps an SDK field name to either "handled" (mapped into the
		// compose model) or "dropped: <reason>" (a conscious omission).
		covered map[string]string
	}{
		{
			name: "ContainerSpec",
			typ:  reflect.TypeOf(swarm.ContainerSpec{}),
			covered: map[string]string{
				"Image":           "handled",
				"Labels":          "handled",
				"Command":         "handled",
				"Args":            "handled",
				"Hostname":        "handled",
				"Env":             "handled",
				"Dir":             "handled",
				"User":            "handled",
				"Init":            "handled",
				"StopSignal":      "handled",
				"ReadOnly":        "handled",
				"Mounts":          "handled",
				"StopGracePeriod": "handled",
				"Healthcheck":     "handled",
				"Hosts":           "handled",
				"DNSConfig":       "handled",
				"Secrets":         "handled",
				"Configs":         "handled",
				"Sysctls":         "handled",
				"CapabilityAdd":   "handled",
				"CapabilityDrop":  "handled",
				"Ulimits":         "handled",
				"Groups":          "dropped: Tier-3, no common compose mapping",
				"Privileges":      "dropped: Tier-3 (cred_spec/selinux/seccomp/apparmor/no_new_privileges)",
				"TTY":             "dropped: Tier-3 (tty)",
				"OpenStdin":       "dropped: Tier-3 (stdin_open)",
				"Isolation":       "dropped: Tier-3 (windows isolation)",
				"OomScoreAdj":     "dropped: Tier-3",
			},
		},
		{
			name: "TaskSpec",
			typ:  reflect.TypeOf(swarm.TaskSpec{}),
			covered: map[string]string{
				"ContainerSpec":         "handled",
				"Resources":             "handled",
				"RestartPolicy":         "handled",
				"Placement":             "handled",
				"Networks":              "handled",
				"LogDriver":             "handled",
				"PluginSpec":            "dropped: non-container runtime",
				"NetworkAttachmentSpec": "dropped: non-container runtime",
				"ForceUpdate":           "dropped: internal counter, not user config",
				"Runtime":               "dropped: non-container runtime",
			},
		},
		{
			name: "ServiceSpec",
			typ:  reflect.TypeOf(swarm.ServiceSpec{}),
			covered: map[string]string{
				"Annotations":    "handled", // embedded: Name + Labels
				"TaskTemplate":   "handled",
				"Mode":           "handled",
				"UpdateConfig":   "handled",
				"RollbackConfig": "handled",
				"Networks":       "handled",
				"EndpointSpec":   "handled",
			},
		},
	}

	for _, tc := range cases {
		for i := 0; i < tc.typ.NumField(); i++ {
			f := tc.typ.Field(i)
			if f.PkgPath != "" {
				continue // unexported
			}
			if _, ok := tc.covered[f.Name]; !ok {
				t.Errorf("swarm.%s.%s is not mapped or waived — map it in "+
					"stack_to_compose.go or add it to the dropped set (#430)",
					tc.name, f.Name)
			}
		}
	}
}
