// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"swarmcli/charts"
)

const chartsUsage = `Usage: swarmcli charts <command> [options]

Repository:
  repo add <name> <url>      Add a chart repository and download its index
  repo list                  List configured repositories
  repo update [name]         Refresh repository indexes (all, or one)
  repo remove <name>         Remove a repository

Discovery:
  search [keyword]           Search charts across repositories
  show chart  <repo/chart>   Show chart metadata
  show values <repo/chart>   Show default values.yaml
  show schema <repo/chart>   Show values.schema.json

Releases:
  template <release> <chart>  Render manifest to stdout (no deploy)
  install  <release> <chart>  Install a chart as a release
  upgrade  <release> <chart>  Upgrade a release to a new revision
  uninstall <release>         Remove a release (keeps volumes)
  rollback <release> <rev>    Re-deploy the contents of a past revision
  history <release>           Show a release's revision history
  get values|manifest <rel>   Show stored values or rendered manifest
  diff upgrade <rel> <chart>  Preview manifest changes before upgrading
  list                        List releases
  status <release>            Show release status and services

Common options:
  -f, --values <file>   Values file (repeatable)
      --set k=v         Override a value (repeatable)
      --version <ver>   Chart version (default: latest)
      --dry-run         Render and validate without deploying
      --wait            Wait for services to converge
      --timeout <dur>   Wait timeout, e.g. 10m (default 5m)
      --history-max <n> Max release revisions to retain
      --install         upgrade: install the release if absent
      --reuse-values    upgrade/diff: layer overrides on previous values
      --revision <n>    get: select a specific revision
      --purge-volumes   uninstall: also remove the release's volumes
`

// chartsMain dispatches `swarmcli charts ...`.
func chartsMain(args []string) int {
	if len(args) == 0 {
		out(chartsUsage)
		return 0
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "-h", "--help", "help":
		out(chartsUsage)
		return 0
	case "repo":
		return chartsRepo(rest)
	case "search":
		return chartsSearch(rest)
	case "show":
		return chartsShow(rest)
	case "template":
		return chartsTemplate(rest)
	case "install":
		return chartsInstall(rest)
	case "upgrade":
		return chartsUpgrade(rest)
	case "uninstall":
		return chartsUninstall(rest)
	case "rollback":
		return chartsRollback(rest)
	case "history":
		return chartsHistory(rest)
	case "get":
		return chartsGet(rest)
	case "diff":
		return chartsDiff(rest)
	case "list", "ls":
		return chartsList(rest)
	case "status":
		return chartsStatus(rest)
	default:
		return usageErr(fmt.Sprintf("unknown charts command %q\n\n%s", sub, chartsUsage))
	}
}

func newStore() (*charts.RepoStore, int) {
	s, err := charts.NewRepoStore()
	if err != nil {
		return nil, fail(err)
	}
	return s, -1
}

// --- repo ---

func chartsRepo(args []string) int {
	if len(args) == 0 {
		return usageErr("charts repo requires a subcommand: add|list|update|remove")
	}
	store, code := newStore()
	if code >= 0 {
		return code
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "add":
		if len(rest) != 2 {
			return usageErr("charts repo add <name> <url>")
		}
		if err := store.Add(rest[0], rest[1]); err != nil {
			return fail(err)
		}
		outf("%q has been added to your repositories\n", rest[0])
		return 0
	case "list", "ls":
		repos, err := store.List()
		if err != nil {
			return fail(err)
		}
		if len(repos) == 0 {
			outln("No repositories configured. Add one with: swarmcli charts repo add <name> <url>")
			return 0
		}
		rows := make([][]string, 0, len(repos))
		for _, r := range repos {
			rows = append(rows, []string{r.Name, r.URL})
		}
		table([]string{"NAME", "URL"}, rows)
		return 0
	case "update":
		name := ""
		if len(rest) == 1 {
			name = rest[0]
		} else if len(rest) > 1 {
			return usageErr("charts repo update [name]")
		}
		changed, unchanged, err := store.Update(name)
		if err != nil {
			return fail(err)
		}
		if len(changed)+len(unchanged) == 0 {
			outln("No repositories to update.")
			return 0
		}
		if len(changed) > 0 {
			outf("Updated %d repositor%s: %s\n", len(changed), plural(len(changed), "y", "ies"), strings.Join(changed, ", "))
		}
		if len(unchanged) > 0 {
			outf("Already up-to-date: %s\n", strings.Join(unchanged, ", "))
		}
		return 0
	case "remove", "rm":
		if len(rest) != 1 {
			return usageErr("charts repo remove <name>")
		}
		if err := store.Remove(rest[0]); err != nil {
			return fail(err)
		}
		outf("%q has been removed from your repositories\n", rest[0])
		return 0
	default:
		return usageErr(fmt.Sprintf("unknown charts repo command %q", sub))
	}
}

// --- search ---

func chartsSearch(args []string) int {
	pos, _, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	keyword := ""
	if len(pos) > 0 {
		keyword = pos[0]
	}
	store, code := newStore()
	if code >= 0 {
		return code
	}
	hits, err := store.Search(keyword)
	if err != nil {
		return fail(err)
	}
	if len(hits) == 0 {
		outln("No charts found.")
		return 0
	}
	rows := make([][]string, 0, len(hits))
	for _, h := range hits {
		rows = append(rows, []string{h.Repo + "/" + h.Entry.Name, h.Entry.Version, h.Entry.AppVersion, h.Entry.Description})
	}
	table([]string{"NAME", "VERSION", "APP VERSION", "DESCRIPTION"}, rows)
	return 0
}

// --- show ---

func chartsShow(args []string) int {
	if len(args) < 2 {
		return usageErr("charts show <chart|values|schema> <repo/chart>")
	}
	what, ref := args[0], args[1]
	pos, f, err := parseArgs(args[2:])
	if err != nil {
		return usageErr(err.Error())
	}
	_ = pos
	ch, _, code := loadChart(ref, f.version)
	if code >= 0 {
		return code
	}
	switch what {
	case "chart":
		printChartMeta(ch)
	case "values":
		data, err := yaml.Marshal(ch.Values)
		if err != nil {
			return fail(err)
		}
		_, _ = stdout.Write(data)
	case "schema":
		if len(ch.Schema) == 0 {
			outln("Chart has no values.schema.json")
			return 0
		}
		_, _ = stdout.Write(ch.Schema)
	default:
		return usageErr(fmt.Sprintf("unknown show target %q (want chart|values|schema)", what))
	}
	return 0
}

func printChartMeta(ch *charts.Chart) {
	m := ch.Metadata
	outf("Name:        %s\n", m.Name)
	outf("Version:     %s\n", m.Version)
	if m.AppVersion != "" {
		outf("App Version: %s\n", m.AppVersion)
	}
	if m.Description != "" {
		outf("Description: %s\n", m.Description)
	}
	for _, mt := range m.Maintainers {
		outf("Maintainer:  %s <%s>\n", mt.Name, mt.Email)
	}
	if ch.Readme != "" {
		outln("\n--- README ---")
		outln(ch.Readme)
	}
}

// --- template / install ---

func chartsTemplate(args []string) int {
	pos, f, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	if len(pos) != 2 {
		return usageErr("charts template <release> <repo/chart>")
	}
	release, ref := pos[0], pos[1]
	manifest, _, _, code := prepare(release, ref, f, nil)
	if code >= 0 {
		return code
	}
	outln(manifest)
	return 0
}

func chartsInstall(args []string) int {
	pos, f, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	if len(pos) != 2 {
		return usageErr("charts install <release> <repo/chart>")
	}
	release, ref := pos[0], pos[1]
	manifest, values, rc, code := prepare(release, ref, f, nil)
	if code >= 0 {
		return code
	}
	rel, err := charts.NewEngine().Install(context.Background(), release, rc, values, manifest, charts.InstallOptions{
		DryRun:     f.dryRun,
		Wait:       f.wait,
		Timeout:    f.timeout,
		HistoryMax: f.historyMax,
	})
	if err != nil {
		return fail(err)
	}
	return reportRelease(rel, manifest, f.dryRun)
}

func chartsUpgrade(args []string) int {
	pos, f, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	if len(pos) != 2 {
		return usageErr("charts upgrade <release> <repo/chart>")
	}
	release, ref := pos[0], pos[1]

	var base map[string]any
	if f.reuseValues {
		cur, err := charts.NewEngine().GetRevision(context.Background(), release, 0)
		switch {
		case err == nil:
			base = cur.Values
		case !f.install:
			return fail(err)
			// With --install and no existing release there are no prior values to
			// reuse; fall through with base=nil so chart defaults apply. Upgrade
			// re-validates the release, so a genuine backend error still surfaces.
		}
	}
	manifest, values, rc, code := prepare(release, ref, f, base)
	if code >= 0 {
		return code
	}
	rel, err := charts.NewEngine().Upgrade(context.Background(), release, rc, values, manifest, charts.InstallOptions{
		DryRun:     f.dryRun,
		Wait:       f.wait,
		Install:    f.install,
		Timeout:    f.timeout,
		HistoryMax: f.historyMax,
	})
	if err != nil {
		return fail(err)
	}
	return reportRelease(rel, manifest, f.dryRun)
}

func reportRelease(rel *charts.Release, manifest string, dryRun bool) int {
	if dryRun {
		outf("NAME: %s\nREVISION: %d\nSTATUS: %s (dry-run, not deployed)\n\n", rel.Name, rel.Revision, rel.Status)
		outln(manifest)
		return 0
	}
	outf("NAME: %s\nREVISION: %d\nSTATUS: %s\nCHART: %s-%s\n", rel.Name, rel.Revision, rel.Status, rel.Chart.Name, rel.Chart.Version)
	return 0
}

func chartsRollback(args []string) int {
	pos, f, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	if len(pos) != 2 {
		return usageErr("charts rollback <release> <revision>")
	}
	rev, err := parseInt(pos[1])
	if err != nil {
		return usageErr(fmt.Sprintf("invalid revision %q", pos[1]))
	}
	rel, err := charts.NewEngine().Rollback(context.Background(), pos[0], rev, charts.InstallOptions{
		Wait: f.wait, Timeout: f.timeout, HistoryMax: f.historyMax,
	})
	if err != nil {
		return fail(err)
	}
	outf("Rolled back %s to the contents of revision %d (new revision %d)\n", rel.Name, rev, rel.Revision)
	return 0
}

func chartsHistory(args []string) int {
	pos, _, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	if len(pos) != 1 {
		return usageErr("charts history <release>")
	}
	hist, err := charts.NewEngine().History(context.Background(), pos[0])
	if err != nil {
		return fail(err)
	}
	rows := make([][]string, 0, len(hist))
	for _, r := range hist {
		rows = append(rows, []string{strconv.Itoa(r.Revision), r.Status, r.Chart.Name + "-" + r.Chart.Version, r.Created})
	}
	table([]string{"REVISION", "STATUS", "CHART", "UPDATED"}, rows)
	return 0
}

func chartsGet(args []string) int {
	if len(args) < 2 {
		return usageErr("charts get <values|manifest> <release> [--revision N]")
	}
	what, release := args[0], args[1]
	_, f, err := parseArgs(args[2:])
	if err != nil {
		return usageErr(err.Error())
	}
	rel, err := charts.NewEngine().GetRevision(context.Background(), release, f.revision)
	if err != nil {
		return fail(err)
	}
	switch what {
	case "values":
		data, err := yaml.Marshal(rel.Values)
		if err != nil {
			return fail(err)
		}
		_, _ = stdout.Write(data)
	case "manifest":
		outln(rel.Manifest)
	default:
		return usageErr(fmt.Sprintf("unknown get target %q (want values|manifest)", what))
	}
	return 0
}

func chartsDiff(args []string) int {
	if len(args) < 1 {
		return usageErr("charts diff upgrade <release> <repo/chart>")
	}
	if args[0] != "upgrade" {
		return usageErr(fmt.Sprintf("unknown diff target %q (want upgrade)", args[0]))
	}
	pos, f, err := parseArgs(args[1:])
	if err != nil {
		return usageErr(err.Error())
	}
	if len(pos) != 2 {
		return usageErr("charts diff upgrade <release> <repo/chart>")
	}
	release, ref := pos[0], pos[1]

	cur, err := charts.NewEngine().GetRevision(context.Background(), release, 0)
	if err != nil {
		return fail(err)
	}
	var base map[string]any
	if f.reuseValues {
		base = cur.Values
	}
	next, _, _, code := prepare(release, ref, f, base)
	if code >= 0 {
		return code
	}
	if cur.Manifest == next {
		outln("No changes.")
		return 0
	}
	out(lineDiff(cur.Manifest, next))
	return 0
}

// prepare loads the chart, merges + validates values, and renders the manifest.
// base, when non-nil, replaces the chart defaults as the merge base (used by
// `upgrade --reuse-values` to layer overrides over the previous release).
func prepare(release, ref string, f flags, base map[string]any) (manifest string, values map[string]any, rc charts.ReleaseChart, code int) {
	ch, _, c := loadChart(ref, f.version)
	if c >= 0 {
		return "", nil, rc, c
	}
	if base == nil {
		base = ch.Values
	}
	files, err := readValuesFiles(f.values)
	if err != nil {
		return "", nil, rc, fail(err)
	}
	values, err = charts.MergeValues(base, files, f.sets)
	if err != nil {
		return "", nil, rc, fail(err)
	}
	if err := charts.ValidateValues(ch.Schema, values); err != nil {
		return "", nil, rc, fail(err)
	}
	manifest, err = charts.Render(ch, charts.RenderContext{
		Values:  values,
		Release: charts.ReleaseMeta{Name: release, Namespace: release, Revision: 1},
		Chart:   charts.ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version, AppVersion: ch.Metadata.AppVersion},
	})
	if err != nil {
		return "", nil, rc, fail(err)
	}
	rc = charts.ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version, AppVersion: ch.Metadata.AppVersion}
	return manifest, values, rc, -1
}

// --- uninstall / list / status ---

func chartsUninstall(args []string) int {
	pos, f, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	if len(pos) != 1 {
		return usageErr("charts uninstall <release> [--purge-volumes]")
	}
	if err := charts.NewEngine().Uninstall(context.Background(), pos[0], f.purge); err != nil {
		return fail(err)
	}
	outf("release %q uninstalled\n", pos[0])
	return 0
}

func chartsList(args []string) int {
	if _, _, err := parseArgs(args); err != nil {
		return usageErr(err.Error())
	}
	rels, err := charts.NewEngine().List(context.Background())
	if err != nil {
		return fail(err)
	}
	if len(rels) == 0 {
		outln("No releases found.")
		return 0
	}
	rows := make([][]string, 0, len(rels))
	for _, r := range rels {
		rows = append(rows, []string{r.Name, strconv.Itoa(r.Revision), r.Status, r.Chart.Name + "-" + r.Chart.Version, r.Created})
	}
	table([]string{"NAME", "REVISION", "STATUS", "CHART", "UPDATED"}, rows)
	return 0
}

func chartsStatus(args []string) int {
	pos, _, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	if len(pos) != 1 {
		return usageErr("charts status <release>")
	}
	rel, svcs, err := charts.NewEngine().Status(context.Background(), pos[0])
	if err != nil {
		return fail(err)
	}
	outf("NAME: %s\nREVISION: %d\nSTATUS: %s\nCHART: %s-%s\nUPDATED: %s\n\n",
		rel.Name, rel.Revision, rel.Status, rel.Chart.Name, rel.Chart.Version, rel.Created)
	if len(svcs) == 0 {
		outln("No services running.")
		return 0
	}
	rows := make([][]string, 0, len(svcs))
	for _, s := range svcs {
		rows = append(rows, []string{s.Name, s.Mode, s.Replicas, s.Status})
	}
	table([]string{"SERVICE", "MODE", "REPLICAS", "STATUS"}, rows)
	return 0
}

// --- chart source resolution ---

// loadChart resolves a chart reference: an existing local directory or .tgz
// path, otherwise a "repo/chart" reference pulled from a configured repository.
func loadChart(ref, version string) (*charts.Chart, charts.ReleaseChart, int) {
	if info, err := os.Stat(ref); err == nil {
		var (
			ch   *charts.Chart
			lerr error
		)
		if info.IsDir() {
			ch, lerr = charts.LoadChartDir(ref)
		} else {
			ch, lerr = loadArchivePath(ref)
		}
		if lerr != nil {
			return nil, charts.ReleaseChart{}, fail(lerr)
		}
		return ch, releaseChartOf(ch), -1
	}

	store, err := charts.NewRepoStore()
	if err != nil {
		return nil, charts.ReleaseChart{}, fail(err)
	}
	entry, base, err := store.Resolve(ref, version)
	if err != nil {
		return nil, charts.ReleaseChart{}, fail(err)
	}
	ch, err := store.Pull(entry, base)
	if err != nil {
		return nil, charts.ReleaseChart{}, fail(err)
	}
	return ch, releaseChartOf(ch), -1
}

func loadArchivePath(path string) (*charts.Chart, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }()
	return charts.LoadChartArchive(fh)
}

func releaseChartOf(ch *charts.Chart) charts.ReleaseChart {
	return charts.ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version, AppVersion: ch.Metadata.AppVersion}
}

func readValuesFiles(paths []string) ([][]byte, error) {
	var out [][]byte
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read values file %q: %w", p, err)
		}
		out = append(out, b)
	}
	return out, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
