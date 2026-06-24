// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/swarm"
)

func TestIsUpdateOutOfSequence(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"daemon message", errors.New("Error response from daemon: rpc error: code = Unknown desc = update out of sequence"), true},
		{"mixed case", errors.New("Update Out Of Sequence"), true},
		{"unrelated", errors.New("connection refused"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isUpdateOutOfSequence(c.err); got != c.want {
				t.Fatalf("isUpdateOutOfSequence(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

// fakeNodeUpdater satisfies nodeUpdater. Each NodeUpdate that fails bumps the
// stored version, so the next NodeInspectWithRaw returns a fresh index —
// modelling the concurrent status writes that cause "update out of sequence".
type fakeNodeUpdater struct {
	curVersion uint64
	spec       swarm.NodeSpec
	inspectErr error
	updateErr  error // returned by NodeUpdate while failsLeft > 0
	failsLeft  int

	inspects int
	updates  int
	lastSpec swarm.NodeSpec
	lastVer  swarm.Version
}

func (f *fakeNodeUpdater) NodeInspectWithRaw(_ context.Context, _ string) (swarm.Node, []byte, error) {
	f.inspects++
	if f.inspectErr != nil {
		return swarm.Node{}, nil, f.inspectErr
	}
	return swarm.Node{
		Meta: swarm.Meta{Version: swarm.Version{Index: f.curVersion}},
		Spec: f.spec,
	}, nil, nil
}

func (f *fakeNodeUpdater) NodeUpdate(_ context.Context, _ string, version swarm.Version, spec swarm.NodeSpec) error {
	f.updates++
	f.lastVer = version
	f.lastSpec = spec
	if f.failsLeft > 0 {
		f.failsLeft--
		f.curVersion++ // a concurrent write moved the index forward
		return f.updateErr
	}
	return nil
}

func TestUpdateNodeSpec_SucceedsFirstTry(t *testing.T) {
	f := &fakeNodeUpdater{curVersion: 7, spec: swarm.NodeSpec{}}

	err := updateNodeSpec(context.Background(), f, "n1", func(spec *swarm.NodeSpec) {
		if spec.Labels == nil {
			spec.Labels = map[string]string{}
		}
		spec.Labels["k"] = "v"
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.inspects != 1 || f.updates != 1 {
		t.Fatalf("inspects=%d updates=%d, want 1/1", f.inspects, f.updates)
	}
	if f.lastVer.Index != 7 {
		t.Fatalf("update used version %d, want 7", f.lastVer.Index)
	}
	if f.lastSpec.Labels["k"] != "v" {
		t.Fatalf("mutate not applied to submitted spec: %#v", f.lastSpec.Labels)
	}
}

func TestUpdateNodeSpec_RetriesThenSucceeds(t *testing.T) {
	f := &fakeNodeUpdater{
		curVersion: 3,
		updateErr:  errors.New("Error response from daemon: update out of sequence"),
		failsLeft:  2,
	}

	err := updateNodeSpec(context.Background(), f, "n1", func(*swarm.NodeSpec) {})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.inspects != 3 || f.updates != 3 {
		t.Fatalf("inspects=%d updates=%d, want 3/3", f.inspects, f.updates)
	}
	if f.lastVer.Index != 5 { // 3 + two concurrent bumps; proves it re-inspected
		t.Fatalf("final update used version %d, want 5 (re-fetched)", f.lastVer.Index)
	}
}

func TestUpdateNodeSpec_GivesUpAfterMaxAttempts(t *testing.T) {
	f := &fakeNodeUpdater{
		updateErr: errors.New("update out of sequence"),
		failsLeft: nodeUpdateMaxAttempts + 1,
	}

	err := updateNodeSpec(context.Background(), f, "n1", func(*swarm.NodeSpec) {})

	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !strings.Contains(err.Error(), "update out of sequence") {
		t.Fatalf("error should wrap the daemon message: %v", err)
	}
	if f.updates != nodeUpdateMaxAttempts {
		t.Fatalf("updates=%d, want %d", f.updates, nodeUpdateMaxAttempts)
	}
}

func TestUpdateNodeSpec_NonRetryableErrorReturnsImmediately(t *testing.T) {
	f := &fakeNodeUpdater{
		updateErr: errors.New("node not found"),
		failsLeft: 1,
	}

	err := updateNodeSpec(context.Background(), f, "n1", func(*swarm.NodeSpec) {})

	if err == nil || !strings.Contains(err.Error(), "node not found") {
		t.Fatalf("want non-retryable error returned verbatim, got %v", err)
	}
	if f.updates != 1 {
		t.Fatalf("updates=%d, want 1 (no retry on non-conflict error)", f.updates)
	}
}

func TestUpdateNodeSpec_InspectErrorIsWrapped(t *testing.T) {
	f := &fakeNodeUpdater{inspectErr: errors.New("boom")}

	err := updateNodeSpec(context.Background(), f, "n1", func(*swarm.NodeSpec) {})

	if err == nil || !strings.Contains(err.Error(), "inspect node") {
		t.Fatalf("want inspect-node wrapped error, got %v", err)
	}
	if f.updates != 0 {
		t.Fatalf("updates=%d, want 0 (no update after inspect failure)", f.updates)
	}
}
