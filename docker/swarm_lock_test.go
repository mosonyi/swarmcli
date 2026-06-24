// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"errors"
	"testing"

	"github.com/docker/docker/api/types/swarm"
)

func TestIsSwarmLockedErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"daemon locked message", errors.New(`Error response from daemon: Swarm is encrypted and needs to be unlocked before it can be used. Please use "docker swarm unlock" to unlock it.`), true},
		{"encrypted only", errors.New("swarm is encrypted"), true},
		{"unlock phrase only", errors.New("the cluster needs to be unlocked"), true},
		{"mixed case", errors.New("SWARM IS ENCRYPTED"), true},
		{"unrelated", errors.New("connection refused"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsSwarmLockedErr(c.err); got != c.want {
				t.Fatalf("IsSwarmLockedErr(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

func TestIsUsableSwarmState(t *testing.T) {
	cases := []struct {
		state swarm.LocalNodeState
		want  bool
	}{
		{swarm.LocalNodeStateActive, true},
		{swarm.LocalNodeStateLocked, true},
		{swarm.LocalNodeStateInactive, false},
		{swarm.LocalNodeStatePending, false},
		{swarm.LocalNodeStateError, false},
		{swarm.LocalNodeState(""), false},
	}
	for _, c := range cases {
		t.Run(string(c.state), func(t *testing.T) {
			if got := isUsableSwarmState(c.state); got != c.want {
				t.Fatalf("isUsableSwarmState(%q) = %v, want %v", c.state, got, c.want)
			}
		})
	}
}
