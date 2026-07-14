// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import "sort"

// OutdatedEntry is one installed release with a newer chart version available.
type OutdatedEntry struct {
	Release   string
	Chart     string
	Repo      string
	Installed string
	Latest    string
}

// Outdated joins installed releases against the newest version of their chart in
// any configured repository index. Releases already at the newest version, and
// those whose chart appears in no index (a local chart), are omitted.
//
// A Release does not record which repository it came from, so a chart present in
// two repositories resolves to the highest version across them, reporting the
// repository that supplied it. That ambiguity is documented rather than designed
// away: recording the source repository is a change to persisted release state,
// which is not worth making for a case most users never hit.
func Outdated(rels []Release, indexes map[string]*Index) []OutdatedEntry {
	var out []OutdatedEntry
	for _, rel := range rels {
		var (
			bestVer  string
			bestRepo string
		)
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
		if bestVer == "" || compareVersions(bestVer, rel.Chart.Version) <= 0 {
			continue
		}
		out = append(out, OutdatedEntry{
			Release:   rel.Name,
			Chart:     rel.Chart.Name,
			Repo:      bestRepo,
			Installed: rel.Chart.Version,
			Latest:    bestVer,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Release < out[j].Release })
	return out
}
