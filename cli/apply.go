// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/Eldara-Tech/swarmcli/charts"
	"github.com/Eldara-Tech/swarmcli/utils/textdiff"
)

// chartsApply converges the swarm to a declarative release manifest.
func chartsApply(args []string) int {
	pos, f, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	// apply's -f names the release manifest, not a values file: per-release values
	// live inside the manifest. A bare positional works too.
	paths := append(append([]string{}, f.values...), pos...)
	if len(paths) != 1 {
		return usageErr("charts apply -f <release-file>")
	}
	if err := rejectUnsupported(f); err != nil {
		return usageErr(err.Error())
	}

	rf, err := charts.LoadReleaseFile(paths[0])
	if err != nil {
		return fail(err)
	}

	store, code := newStore()
	if code >= 0 {
		return code
	}
	if err := store.EnsureRepos(rf.Repositories); err != nil {
		return fail(err)
	}

	e := charts.NewEngine()
	plan, err := e.PlanApply(context.Background(), rf, charts.NewChartSource(store))
	if err != nil {
		return fail(err)
	}

	printPlan(plan, f.diff)

	// Gate the whole plan before converging any of it, matching PlanApply: a
	// release that cannot run on this build must not leave the swarm half
	// converged. Unchanged releases are exempt — they are already deployed, and
	// apply is not going to touch them.
	//
	// The preview verbs only report, so they warn; the rest refuse without ever
	// prompting, because apply is meant to run unattended and must not block on
	// stdin just because a terminal happens to be attached.
	pol := compatEnforceNoPrompt
	if f.dryRun || f.diff {
		pol = compatWarn
	}
	for _, r := range plan.Releases {
		if r.Action == charts.ActionUnchanged {
			continue
		}
		if code := applyCompat(r.Compat, pol, f.skipCompatCheck); code >= 0 {
			return code
		}
	}

	// --diff is a preview verb; it must never deploy.
	if f.dryRun || f.diff {
		outln("\ndry-run: nothing was deployed")
		return 0
	}

	install, upgrade, _ := plan.Counts()
	if install+upgrade == 0 {
		reportUnmanaged(plan)
		return 0
	}

	results, err := e.Apply(context.Background(), plan, charts.InstallOptions{
		Wait:       f.wait,
		Timeout:    f.timeout,
		HistoryMax: f.historyMax,
	})
	// Report what did happen before surfacing the failure: a partial apply has
	// still changed the swarm, and the operator needs to know how far it got.
	for _, r := range results {
		if r.Action == charts.ActionUnchanged {
			continue
		}
		outf("%s %s (revision %d)\n", r.Name, r.Action, r.Revision)
	}
	if err != nil {
		return fail(err)
	}
	reportUnmanaged(plan)
	return 0
}

// rejectUnsupported fails on flags apply does not honour. The charts flag set is
// global, so every subcommand parses every flag and silently ignores whatever it
// does not read. For a command whose entire contract is "the file is the only
// source of truth", quietly discarding --set or --version would be a correctness
// bug, not a cosmetic one.
func rejectUnsupported(f flags) error {
	var bad []string
	if len(f.sets) > 0 {
		bad = append(bad, "--set")
	}
	if f.version != "" {
		bad = append(bad, "--version")
	}
	if f.reuseValues {
		bad = append(bad, "--reuse-values")
	}
	if f.install {
		bad = append(bad, "--install")
	}
	if f.purge {
		bad = append(bad, "--purge-volumes")
	}
	if f.requirements {
		bad = append(bad, "--requirements")
	}
	if f.revision != 0 {
		bad = append(bad, "--revision")
	}
	if len(bad) == 0 {
		return nil
	}
	return fmt.Errorf("%s not supported by apply: the release file is the only source of truth "+
		"(set chart versions and values there)", strings.Join(bad, ", "))
}

func printPlan(plan *charts.Plan, withDiff bool) {
	var rows [][]string
	for _, r := range plan.Releases {
		from := r.FromVersion
		if from == "" {
			from = "-"
		}
		rows = append(rows, []string{r.Name, r.Ref, from, r.ToVersion, string(r.Action)})
	}
	table([]string{"RELEASE", "CHART", "FROM", "TO", "ACTION"}, rows)

	if withDiff {
		for _, r := range plan.Releases {
			if r.Action == charts.ActionUnchanged {
				continue
			}
			outf("\n--- %s (%s) ---\n", r.Name, r.Action)
			out(textdiff.Lines(r.CurrentManifest, r.Manifest))
		}
	}

	install, upgrade, unchanged := plan.Counts()
	outf("\n%d to install, %d to upgrade, %d unchanged\n", install, upgrade, unchanged)
}

// reportUnmanaged names releases on the swarm that the file does not describe.
// apply never removes them: a release records nothing about which manifest
// produced it, so there is no way to tell one owned by a second file, or one
// installed by hand, from a genuinely obsolete one.
func reportUnmanaged(plan *charts.Plan) {
	if len(plan.Unmanaged) == 0 {
		return
	}
	outf("\n%d release(s) exist on this swarm but are not in the release file:\n", len(plan.Unmanaged))
	for _, n := range plan.Unmanaged {
		outf("  %s\n", n)
	}
	out("apply does not remove them; uninstall explicitly if they are obsolete:\n")
	for _, n := range plan.Unmanaged {
		outf("  swarmcli charts uninstall %s\n", n)
	}
}

// chartsOutdated compares every installed release against the newest chart
// version its repositories offer. It is the human-facing complement to an
// automated updater watching the release file.
func chartsOutdated(args []string) int {
	if _, _, err := parseArgs(args); err != nil {
		return usageErr(err.Error())
	}
	store, code := newStore()
	if code >= 0 {
		return code
	}
	// A read-only informational command must degrade, not fail: report against the
	// cached indexes when the network is unavailable.
	if _, _, err := store.Update(""); err != nil {
		errf("could not refresh repositories (%v); comparing against the cached indexes\n", err)
	}
	indexes, err := store.Indexes()
	if err != nil {
		return fail(err)
	}
	rels, err := charts.NewEngine().List(context.Background())
	if err != nil {
		return fail(err)
	}

	entries := charts.Outdated(rels, indexes)
	if len(entries) == 0 {
		outln("All releases are up to date.")
		return 0
	}
	var rows [][]string
	for _, e := range entries {
		rows = append(rows, []string{e.Release, e.Chart, e.Repo, e.Installed, e.Latest})
	}
	table([]string{"RELEASE", "CHART", "REPO", "INSTALLED", "LATEST"}, rows)
	outf("\n%d of %d release(s) outdated.\n", len(entries), len(rels))
	return 0
}
