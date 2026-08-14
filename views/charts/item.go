// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"time"

	"github.com/Eldara-Tech/swarmcli/charts"
)

// releaseItem is one row: every stored revision of a release, plus the live
// state the records cannot tell you.
//
// The revisions are held rather than flattened into display strings because
// inspecting a manifest or values reads them straight off the record — the
// same read that produced the row, so nothing is fetched twice.
type releaseItem struct {
	Name string
	// Revisions is ascending and never empty for a constructed item.
	Revisions []charts.Release
	// Created is the current revision's timestamp, parsed once at load; the
	// zero value when it was missing or malformed.
	Created time.Time

	// Health is what the swarm is doing now, and HasHealth reports whether it
	// applies. It and the stored status disagree exactly when it matters — a
	// release reading deployed whose rollout is wedged.
	Health    charts.Convergence
	HasHealth bool

	// Services backs the health rollup.
	Services []charts.ServiceState

	// Latest is the newest version of this chart in a cached repository index,
	// when that is newer than what is installed. Empty otherwise, which covers
	// both "already current" and "this chart is in no index" — a local chart
	// has nothing to be outdated against.
	Latest string
}

func (i releaseItem) FilterValue() string { return i.Name }

// current is the highest stored revision. The zero Release for an item with no
// revisions, which readReleases never produces but a zero-value item is.
func (i releaseItem) current() charts.Release {
	if len(i.Revisions) == 0 {
		return charts.Release{}
	}
	return i.Revisions[len(i.Revisions)-1]
}

func (i releaseItem) revision() int  { return i.current().Revision }
func (i releaseItem) status() string { return i.current().Status }

// chartRef is "name-version", as `charts list` prints it.
func (i releaseItem) chartRef() string {
	c := i.current().Chart
	if c.Name == "" && c.Version == "" {
		return "—"
	}
	return c.Name + "-" + c.Version
}

// health derives the convergence phase for a release's current revision.
//
// Only a deployed record gets one. Rollup on no services reports "progressing:
// no services are running yet", which is the right answer for a --wait deploy
// and a wrong one on a browser row for a release that failed or never
// installed — those have no rollout in flight to be progressing through.
func health(rel charts.Release, svcs []charts.ServiceState) (charts.Convergence, bool) {
	if rel.Status != charts.StatusDeployed {
		return charts.Convergence{}, false
	}
	return charts.Rollup(svcs), true
}

// healthLabel is the HEALTH cell.
func (i releaseItem) healthLabel() string {
	if !i.HasHealth {
		return "—"
	}
	return string(i.Health.Phase)
}

// healthRank orders the HEALTH column so that ascending is worst-first: a
// wedged release is the one worth looking at, and sorting it to the bottom
// would be the wrong default for the first press of the sort key.
func (i releaseItem) healthRank() int {
	if !i.HasHealth {
		return 3
	}
	switch i.Health.Phase {
	case charts.PhaseWedged:
		return 0
	case charts.PhaseProgressing:
		return 1
	default:
		return 2
	}
}
