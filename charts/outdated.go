// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import "sort"

// Availability is what the cached repository indexes know about one installed
// release's chart. It exists because "nothing newer" and "nothing to compare
// against" are different answers that an absent entry ran together: a local
// chart is in no index, and a release at the newest published version is in one
// — the second is up to date, the first is merely unknowable.
type Availability struct {
	// Repo is the repository that supplied Latest.
	Repo string
	// Latest is the newest version of this chart in any index, whether or not
	// it is newer than what is installed.
	Latest string
	// Newer reports that Latest is an upgrade from the installed version. A
	// release ahead of every index — installed from a local chart, say — is
	// current rather than outdated, so this is false there too.
	Newer bool
}

// Available joins installed releases against the cached repository indexes,
// keyed by release name. A release whose chart appears in no index is absent
// from the result, which is the one fact Outdated cannot report.
//
// A Release does not record which repository it came from, so a chart present in
// two repositories resolves to the highest version across them, reporting the
// repository that supplied it. That ambiguity is documented rather than designed
// away: recording the source repository is a change to persisted release state,
// which is not worth making for a case most users never hit.
func Available(rels []Release, indexes map[string]*Index) map[string]Availability {
	out := make(map[string]Availability, len(rels))
	for _, rel := range rels {
		var bestVer, bestRepo string
		for repo, idx := range indexes {
			versions := idx.Entries[rel.Chart.Name]
			if len(versions) == 0 {
				continue
			}
			latest := latestVersion(versions)
			if bestVer == "" || compareVersions(latest.Version, bestVer) > 0 {
				bestVer, bestRepo = latest.Version, repo
			}
		}
		if bestVer == "" {
			continue
		}
		out[rel.Name] = Availability{
			Repo:   bestRepo,
			Latest: bestVer,
			Newer:  compareVersions(bestVer, rel.Chart.Version) > 0,
		}
	}
	return out
}

// OutdatedEntry is one installed release with a newer chart version available.
type OutdatedEntry struct {
	Release   string
	Chart     string
	Repo      string
	Installed string
	Latest    string
}

// Outdated is Available filtered to the releases with an upgrade waiting.
// Releases already at the newest version, those ahead of every index, and those
// whose chart appears in no index (a local chart) are all omitted — a caller
// that needs to tell those apart wants Available.
func Outdated(rels []Release, indexes map[string]*Index) []OutdatedEntry {
	available := Available(rels, indexes)
	var out []OutdatedEntry
	for _, rel := range rels {
		avail, ok := available[rel.Name]
		if !ok || !avail.Newer {
			continue
		}
		out = append(out, OutdatedEntry{
			Release:   rel.Name,
			Chart:     rel.Chart.Name,
			Repo:      avail.Repo,
			Installed: rel.Chart.Version,
			Latest:    avail.Latest,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Release < out[j].Release })
	return out
}
