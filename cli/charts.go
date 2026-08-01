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

	"github.com/Eldara-Tech/swarmcli/charts"
	"github.com/Eldara-Tech/swarmcli/utils/textdiff"
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

Authoring:
  lint <chart>                Check a chart without deploying it

Releases:
  template <release> <chart>  Render manifest to stdout (no deploy)
  install  <release> <chart>  Install a chart as a release
  upgrade  <release> <chart>  Upgrade a release to a new revision
  uninstall <release>         Remove a release (keeps volumes)
  rollback <release> <rev>    Re-deploy the contents of a past revision
  history <release>           Show a release's revision history
  prune [release]             Delete old revisions beyond --history-max
  get values|manifest <rel>   Show stored values or rendered manifest
  diff upgrade <rel> <chart>  Preview manifest changes before upgrading
  list                        List releases
  status <release>            Show release status and services

GitOps:
  apply -f <file>             Converge the swarm to a declarative release file
  outdated                    Show releases with a newer chart version available

A release file pins each release to a chart version, so it is reproducible and an
automated updater (e.g. Renovate) has something concrete to bump. apply installs
what is missing, upgrades what changed, and skips what already matches. It never
removes a release the file does not mention — it reports those instead.

  # swarmcli-release.yaml
  repositories:
    - name: swarmcli-charts
      url: https://eldara-tech.github.io/swarmcli-charts
  releases:
    - name: edge
      chart: swarmcli-charts/traefik
      version: "0.1.1"
      values: [./traefik.yaml]     # relative to the release file, not the CWD

A chart may declare its external networks/secrets/configs in requirements.yaml:
install pre-flights them (auto-creating networks marked autoCreate, validating
the rest) and uninstall reports any auto-created networks it leaves in place.

A chart may also declare the swarmcli it needs, as a SemVer constraint:

  # Chart.yaml
  swarmcliVersion: ">= 1.13.0"

install, upgrade and apply refuse a chart this build is too old for, so the
failure names the version to upgrade to instead of surfacing as whatever error
the missing feature happens to produce. install and upgrade ask first when run
interactively; apply never does. template, diff and show only warn. Pass
--skip-compat-check to try anyway. Charts declaring nothing are unaffected.

Common options:
  -f, --values <file>   Values file (repeatable). For apply: the release file.
      --set k=v         Override a value (repeatable)
      --version <ver>   Chart version, for a <repo>/<chart> reference (default: latest).
                        Not valid with a local chart path — its Chart.yaml sets the version.
      --dry-run         Render and validate without deploying
      --requirements    template: emit rendered requirements.yaml, not the manifest
      --wait            Wait for services to converge
      --timeout <dur>   Wait timeout, e.g. 10m (default 5m)
      --history-max <n> Max release revisions to retain
      --resolve-image <mode>  always | changed | never — how the daemon resolves
                        image tags to digests at deploy time (default: always)
      --install         upgrade: install the release if absent
      --reuse-values    upgrade/diff: layer overrides on previous values
      --revision <n>    get: select a specific revision
      --purge-volumes   uninstall: also remove the release's volumes
      --diff            apply: show each changed release's manifest diff (implies --dry-run)
      --skip-compat-check  Proceed despite a chart's unmet swarmcliVersion constraint
      --for-version <ver>  lint: check the chart's swarmcliVersion against <ver>
                        instead of this build's chart engine

lint renders a chart and reports every problem it finds: a broken template,
values that fail values.schema.json, a swarmcliVersion this build does not
satisfy. It renders from the chart defaults, layering any -f/--set on top — a
chart with a required, undefaulted input needs them supplied, exactly as an
install would. --for-version asks whether the chart's declared floor admits some
other version — it cannot tell you the chart RUNS on that version, because this
binary carries only its own engine's behaviour. Rendering with a real binary of
that version is the only thing that settles that.

apply honours --wait, --timeout, --history-max and --resolve-image. It REJECTS --set, --version,
--reuse-values, --install, --purge-volumes, --requirements and --revision rather
than ignoring them: the release file is the only source of truth, so a value passed
on the command line would be a lie. outdated refreshes the repository indexes first
and falls back to the cached ones if the network is unavailable.
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
	case "lint":
		return chartsLint(rest)
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
	case "prune":
		return chartsPrune(rest)
	case "get":
		return chartsGet(rest)
	case "diff":
		return chartsDiff(rest)
	case "list", "ls":
		return chartsList(rest)
	case "status":
		return chartsStatus(rest)
	case "apply":
		return chartsApply(rest)
	case "outdated":
		return chartsOutdated(rest)
	default:
		return usageErr(fmt.Sprintf("unknown charts command %q\n\n%s", sub, chartsUsage))
	}
}

func newStore() (*charts.RepoStore, int) {
	s, err := charts.NewRepoStore()
	if err != nil {
		return nil, fail(err)
	}
	s.Warnf = errf
	s.AllowPlaintext = plaintextAllowed()
	return s, -1
}

// plaintextAllowed reports whether the operator opted this machine out of the
// store's https-only default. It is an environment variable rather than a flag
// because a plaintext repository is not only reachable through `repo add`: once
// it is in repos.json, `repo update`, `search`, `install`, `upgrade` and `apply`
// all fetch from it, and a flag would have to be threaded — honestly — through
// every one of them. Someone running an internal registry sets this once, in a
// shell profile or a CI job, and everything keeps working.
func plaintextAllowed() bool {
	allowed, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(charts.AllowPlaintextEnv)))
	return err == nil && allowed
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
	if code := applyCompat(charts.CheckCompat(ch.Metadata), compatWarn, f.skipCompatCheck); code >= 0 {
		return code
	}
	switch what {
	case "chart":
		printChartMeta(ch)
	case "values":
		// Print the chart's values.yaml verbatim so its comments and key order
		// survive (re-marshalling the parsed map would drop both). Fall back to
		// marshalling for a chart that ships no values.yaml.
		if len(ch.ValuesRaw) > 0 {
			_, _ = stdout.Write(ch.ValuesRaw)
			break
		}
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

// chartsLint checks a chart without deploying anything: it is what a chart
// author runs before publishing, and what a chart repository's CI runs per
// chart. -f/--set supply the values the render check needs — a chart with a
// required, undefaulted input cannot render from bare defaults.
//
// --for-version lints against a chart-engine version other than this build's,
// which answers "does this chart's declared floor admit X?". It cannot answer
// "does this chart run on X" — only a real X can (see charts.CheckCompatAgainst).
func chartsLint(args []string) int {
	pos, f, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	if len(pos) != 1 {
		return usageErr("charts lint <chart>")
	}
	ch, _, code := loadChart(pos[0], f.version)
	if code >= 0 {
		return code
	}
	files, err := readValuesFiles(f.values)
	if err != nil {
		return fail(err)
	}

	engine := charts.EngineVersion()
	if f.forVersion != "" {
		engine = f.forVersion
	}
	findings := charts.Lint(ch, engine, files, f.sets)
	for _, fd := range findings {
		errf("%s: %s\n", fd.Severity, fd.Message)
	}
	if charts.HasErrors(findings) {
		return 1
	}
	outf("%s %s: ok\n", ch.Metadata.Name, ch.Metadata.Version)
	return 0
}

func chartsTemplate(args []string) int {
	pos, f, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	if len(pos) != 2 {
		return usageErr("charts template <release> <repo/chart>")
	}
	release, ref := pos[0], pos[1]
	manifest, _, _, req, _, code := prepare(release, ref, f, nil, compatWarn)
	if code >= 0 {
		return code
	}
	// --requirements emits the chart's requirements.yaml rendered with the same
	// values as the manifest (the resolved external-resource contract), instead of
	// the manifest. A chart with no requirements.yaml emits nothing.
	if f.requirements {
		if req == nil {
			return 0
		}
		out, err := yaml.Marshal(req)
		if err != nil {
			return fail(err)
		}
		outln(strings.TrimRight(string(out), "\n"))
		return 0
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
	manifest, values, rc, req, chartFiles, code := prepare(release, ref, f, nil, compatEnforce)
	if code >= 0 {
		return code
	}
	rel, err := charts.NewEngine().Install(context.Background(), release, rc, values, manifest, charts.InstallOptions{
		DryRun:       f.dryRun,
		Wait:         f.wait,
		Timeout:      f.timeout,
		HistoryMax:   f.historyMax,
		Requirements: req,
		Files:        chartFiles,
		ResolveImage: f.resolveImage,
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
	manifest, values, rc, req, chartFiles, code := prepare(release, ref, f, base, compatEnforce)
	if code >= 0 {
		return code
	}
	rel, err := charts.NewEngine().Upgrade(context.Background(), release, rc, values, manifest, charts.InstallOptions{
		DryRun:       f.dryRun,
		Wait:         f.wait,
		Install:      f.install,
		Timeout:      f.timeout,
		HistoryMax:   f.historyMax,
		Requirements: req,
		Files:        chartFiles,
		ResolveImage: f.resolveImage,
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

func chartsPrune(args []string) int {
	pos, f, err := parseArgs(args)
	if err != nil {
		return usageErr(err.Error())
	}
	if len(pos) > 1 {
		return usageErr("charts prune [release] [--history-max <n>] [--dry-run]")
	}
	e := charts.NewEngine()
	ctx := context.Background()

	var results []charts.PruneResult
	if len(pos) == 1 {
		var res charts.PruneResult
		res, err = e.Prune(ctx, pos[0], f.historyMax, f.dryRun)
		results = []charts.PruneResult{res}
	} else {
		results, err = e.PruneAll(ctx, f.historyMax, f.dryRun)
	}
	if err != nil {
		return fail(err)
	}

	if f.historyMax <= 0 {
		errf("no --history-max retention window given; all revisions kept (pass --history-max <n> to prune)\n")
		return 0
	}

	rows := make([][]string, 0)
	deleted := 0
	for _, res := range results {
		for _, a := range res.Actions {
			action := "keep"
			switch {
			case a.Delete:
				action = "delete"
				deleted++
			case a.Current:
				action = "keep (current)"
			}
			rows = append(rows, []string{res.Release, strconv.Itoa(a.Revision), action})
		}
	}
	table([]string{"RELEASE", "REVISION", "ACTION"}, rows)

	if f.dryRun {
		outf("dry-run: %d revision(s) would be pruned\n", deleted)
	} else {
		outf("pruned %d revision(s)\n", deleted)
	}
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
	next, _, _, _, _, code := prepare(release, ref, f, base, compatWarn)
	if code >= 0 {
		return code
	}
	if cur.Manifest == next {
		outln("No changes.")
		return 0
	}
	out(textdiff.Lines(cur.Manifest, next))
	return 0
}

// prepare loads the chart, merges + validates values, and renders the manifest.
// base, when non-nil, replaces the chart defaults as the merge base (used by
// `upgrade --reuse-values` to layer overrides over the previous release).
//
// chartFiles are the chart files the rendered manifest's file:/env_file: keys
// name. They are resolved here because this is one of only two places a *Chart
// and a rendered manifest coexist — the chart is gone by the time any of this
// returns — and every command routed through prepare wants the same answer:
// template and diff show a manifest that could actually be deployed, install
// and upgrade deploy it.
func prepare(release, ref string, f flags, base map[string]any, pol compatPolicy) (manifest string, values map[string]any, rc charts.ReleaseChart, req *charts.Requirements, chartFiles map[string][]byte, code int) {
	ch, _, c := loadChart(ref, f.version)
	if c >= 0 {
		return "", nil, rc, nil, nil, c
	}
	// Before rendering, not after: a chart that needs a newer engine would
	// otherwise fail somewhere inside Render, and "function X not defined" is a
	// far worse answer than naming the version it wants.
	if c := applyCompat(charts.CheckCompat(ch.Metadata), pol, f.skipCompatCheck); c >= 0 {
		return "", nil, rc, nil, nil, c
	}
	if base == nil {
		base = ch.Values
	}
	files, err := readValuesFiles(f.values)
	if err != nil {
		return "", nil, rc, nil, nil, fail(err)
	}
	values, err = charts.MergeValues(base, files, f.sets)
	if err != nil {
		return "", nil, rc, nil, nil, fail(err)
	}
	if err := charts.ValidateValues(ch.Schema, values); err != nil {
		return "", nil, rc, nil, nil, fail(err)
	}
	ctx := charts.RenderContext{
		Values:  values,
		Release: charts.ReleaseMeta{Name: release, Namespace: release, Revision: 1},
		Chart:   charts.ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version, AppVersion: ch.Metadata.AppVersion},
	}
	manifest, err = charts.Render(ch, ctx)
	if err != nil {
		return "", nil, rc, nil, nil, fail(err)
	}
	// requirements.yaml is rendered with the same values as the manifest, so an
	// operator-chosen network/secret name (e.g. database.network) is validated
	// against the name the manifest actually references.
	req, err = charts.RenderRequirements(ch, ctx)
	if err != nil {
		return "", nil, rc, nil, nil, fail(err)
	}
	// fail() prints the error and nothing else: a path a chart may not read is a
	// breaking change with no compatibility flag behind it, so this message is
	// the whole of the migration path and must reach the operator verbatim.
	chartFiles, err = charts.ResolveManifestFiles(manifest, ch.Files)
	if err != nil {
		return "", nil, rc, nil, nil, fail(err)
	}
	rc = charts.ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version, AppVersion: ch.Metadata.AppVersion}
	return manifest, values, rc, req, chartFiles, -1
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
	res, err := charts.NewEngine().Uninstall(context.Background(), pos[0], f.purge)
	if err != nil {
		return fail(err)
	}
	outf("release %q uninstalled\n", pos[0])
	if res != nil && len(res.OrphanedNetworks) > 0 {
		out("\nswarmcli auto-created the following external network(s) on install and left\n" +
			"them in place (they may be shared with other stacks):\n")
		for _, n := range res.OrphanedNetworks {
			outf("  %s\n", n)
		}
		out("remove any you no longer need:\n")
		for _, n := range res.OrphanedNetworks {
			outf("  docker network rm %s\n", n)
		}
	}
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
// The resolution itself lives in charts.ChartSource, so apply and the imperative
// commands cannot drift apart.
func loadChart(ref, version string) (*charts.Chart, charts.ReleaseChart, int) {
	store, code := newStore()
	if code >= 0 {
		return nil, charts.ReleaseChart{}, code
	}
	ch, err := charts.NewChartSource(store).Load(ref, version)
	if err != nil {
		return nil, charts.ReleaseChart{}, fail(err)
	}
	return ch, charts.ReleaseChartOf(ch), -1
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
