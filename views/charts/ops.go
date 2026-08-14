// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"context"

	"github.com/Eldara-Tech/swarmcli/charts"
)

// releaseOps is the read-only slice of the release engine this view needs.
//
// It is declared here rather than added to docker.Deps alongside the other op
// interfaces because charts imports docker: a docker.ChartOps returning a
// charts.Release would close an import cycle. Declaring it at the consumer is
// the idiomatic placement anyway, and the view is mocked by assigning this
// field directly.
type releaseOps interface {
	// AllRevisions returns every stored revision grouped by release name,
	// ascending. One call serves both the list and any expanded release's
	// history; List followed by a History per release would decode every
	// stored revision twice.
	AllRevisions(ctx context.Context) (map[string][]charts.Release, error)
	// ServiceStates returns the live state of a release's services. It reads
	// the shared Docker snapshot cache, so calling it once per release costs
	// one snapshot and N in-memory filters.
	ServiceStates(ctx context.Context, release string) []charts.ServiceState
}

// engineOps binds releaseOps to the ambient release engine.
//
// Ambient is deliberate: NewEngine goes through the shared Docker client and
// snapshot cache, both of which the TUI's context switcher already resets, so
// the view follows the active context with no extra wiring. Pinning a context
// name would bypass that cache and re-snapshot on every read.
type engineOps struct{ engine *charts.Engine }

func newEngineOps() releaseOps { return engineOps{engine: charts.NewEngine()} }

func (o engineOps) AllRevisions(ctx context.Context) (map[string][]charts.Release, error) {
	return o.engine.AllRevisions(ctx)
}

func (o engineOps) ServiceStates(ctx context.Context, release string) []charts.ServiceState {
	return o.engine.Backend.StackServices(ctx, release)
}
