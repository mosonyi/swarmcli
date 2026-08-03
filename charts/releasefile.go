// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ReleaseFile is a declarative release manifest: the desired set of releases on a
// swarm, plus the repositories their charts come from. It is the GitOps entry
// point — `charts install` and `charts upgrade` are imperative and release state
// lives in the swarm, so without this file there is nothing in git for an
// automated updater (Renovate, Dependabot, a bot of your own) to edit.
//
// The key names deliberately mirror Helmfile's. Renovate ships a `helmfile`
// manager that reads exactly `repositories[].{name,url}` and
// `releases[].{name,chart,version}`, so pointing it at this file needs one line
// of config and no hand-written regex:
//
//	{"helmfile": {"managerFilePatterns": ["/(^|/)swarmcli-release\\.ya?ml$/"]}}
//
// Unknown keys are a hard error (see ParseReleaseFile), which also contains the
// obvious hazard of borrowing another tool's vocabulary: pasting real Helmfile
// syntax fails loudly and names the key, rather than silently doing half of what
// was meant.
//
// `wave` is the one key that is ours rather than Helmfile's. Renovate's manager
// reads the three keys above and ignores everything else, so declaring it costs
// nothing there; what it does cost is that a file using it cannot be read by a
// swarmcli older than the release that added it, since unknown keys are refused
// rather than skipped. That is true of every key this file has ever gained.
type ReleaseFile struct {
	APIVersion   string     `yaml:"apiVersion,omitempty"`
	Repositories []RepoSpec `yaml:"repositories,omitempty"`
	// Owner names this manifest, claiming every release it installs. It is
	// optional and there is deliberately no default: a derived one would either
	// change between a laptop and a CI checkout (a path hash) or be shared by
	// every repository using the conventional filename (a basename), and either
	// would let two unrelated manifests claim each other's releases. Absent
	// means nothing is claimed, which is exactly today's behaviour.
	Owner    string        `yaml:"owner,omitempty"`
	Releases []ReleaseSpec `yaml:"releases"`

	// Dir is the directory containing the file. Values files and local chart
	// paths resolve against it, never the process working directory, so the
	// manifest is relocatable: a CI job gets the same result no matter where it
	// invoked swarmcli from.
	Dir string `yaml:"-"`
	// Path is the file as given, used to prefix error messages.
	Path string `yaml:"-"`
}

// RepoSpec is a chart repository the file depends on.
type RepoSpec struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// ReleaseSpec is one desired release.
type ReleaseSpec struct {
	Name    string   `yaml:"name"`
	Chart   string   `yaml:"chart"`
	Version string   `yaml:"version,omitempty"`
	Values  []string `yaml:"values,omitempty"`
	// Wave groups releases that are applied together. Every release in one wave
	// is deployed, the whole wave is waited for, and only then does the next
	// start — so a migration that does not converge stops the releases that
	// depend on it from ever being deployed.
	//
	// Ascending, and 0 by default. That is Argo CD's sync-wave, and it is also
	// the reading that keeps a file declaring nothing behaving exactly as it
	// always has: one wave, applied in file order, with no barrier and no extra
	// read of the swarm. Negative is legal, and is how something is put in front
	// of an existing set without renumbering it.
	//
	// Within a wave the order is the file's, and it is not meaningful — that is
	// the whole point of putting two releases in one wave. Do not rely on it.
	Wave int `yaml:"wave,omitempty"`
}

// LoadReleaseFile reads and validates a release manifest.
func LoadReleaseFile(path string) (*ReleaseFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseReleaseFile(data, path)
}

// ParseReleaseFile decodes and validates a release manifest. Unknown keys are
// rejected: this is a file an automated updater rewrites, so a misspelled
// `version:` must fail loudly rather than silently leave the release floating.
func ParseReleaseFile(data []byte, path string) (*ReleaseFile, error) {
	rf := &ReleaseFile{Path: path, Dir: filepath.Dir(path)}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(rf); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := rf.validate(); err != nil {
		return nil, err
	}
	return rf, nil
}

func (rf *ReleaseFile) validate() error {
	if rf.APIVersion != "" && rf.APIVersion != "v1" {
		return rf.errf("unsupported apiVersion '%s' (expected v1)", rf.APIVersion)
	}
	if len(rf.Releases) == 0 {
		return rf.errf("no releases declared")
	}
	if rf.Owner != "" {
		if err := validateOwnerID(rf.Owner); err != nil {
			return rf.errf("%v", err)
		}
	}

	seenRepo := map[string]bool{}
	for i, r := range rf.Repositories {
		// A repository name reaches the store as a path component of its cached
		// index, so it gets the same charset check a release name gets. Checking
		// here as well as in the store is the point: this file may have been
		// written by whoever raised the pull request.
		if err := validateRepoName(r.Name); err != nil {
			return rf.errf("repositories[%d]: %v", i, err)
		}
		switch {
		case r.URL == "":
			return rf.errf("repositories[%d] (%s): url is required", i, r.Name)
		case seenRepo[r.Name]:
			return rf.errf("repositories[%d]: duplicate repository '%s'", i, r.Name)
		}
		seenRepo[r.Name] = true
	}

	seenRelease := map[string]bool{}
	for i, r := range rf.Releases {
		where := fmt.Sprintf("releases[%d]", i)
		if r.Name != "" {
			where = fmt.Sprintf("releases[%d] (%s)", i, r.Name)
		}
		if r.Name == "" {
			return rf.errf("%s: name is required", where)
		}
		if err := validateReleaseName(r.Name); err != nil {
			return rf.errf("%s: %v", where, err)
		}
		if seenRelease[r.Name] {
			return rf.errf("%s: duplicate release '%s'", where, r.Name)
		}
		seenRelease[r.Name] = true

		if r.Chart == "" {
			return rf.errf("%s: chart is required", where)
		}
		if IsPathRef(r.Chart) {
			// A local chart carries its version in its own Chart.yaml; there is
			// nothing for `version:` to select, so accepting it would be a lie.
			if r.Version != "" {
				return rf.errf("%s: version must be omitted for the local chart '%s' (its Chart.yaml sets the version)", where, r.Chart)
			}
			continue
		}
		// The entire point of this file is to be reproducible and to give an
		// updater something concrete to bump. A floating version would silently
		// upgrade a swarm the next time upstream published.
		if r.Version == "" {
			return rf.errf("%s: version is required — a release manifest must be reproducible; run `swarmcli charts search %s` and pin one", where, chartNameOf(r.Chart))
		}
	}
	return nil
}

func (rf *ReleaseFile) errf(format string, a ...any) error {
	return fmt.Errorf("%s: %s", rf.Path, fmt.Sprintf(format, a...))
}

// ownerID is the owner id stamped on releases this file produces: its declared
// owner namespaced under "apply/", so a manifest applied from the command line
// and an application reconciled by a controller cannot claim each other's
// releases by happening to pick the same name. Empty when no owner is declared.
func (rf *ReleaseFile) ownerID() string {
	if rf.Owner == "" {
		return ""
	}
	return "apply/" + rf.Owner
}

// ValuesPaths resolves a release's values files against the manifest's directory.
func (rf *ReleaseFile) ValuesPaths(r ReleaseSpec) []string {
	out := make([]string, 0, len(r.Values))
	for _, p := range r.Values {
		out = append(out, rf.resolve(p))
	}
	return out
}

// ChartRef resolves a release's chart reference, joining local paths against the
// manifest's directory and passing "<repo>/<chart>" references through.
func (rf *ReleaseFile) ChartRef(r ReleaseSpec) string {
	if IsPathRef(r.Chart) {
		return rf.resolve(r.Chart)
	}
	return r.Chart
}

func (rf *ReleaseFile) resolve(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(rf.Dir, p)
}

// chartNameOf returns the chart part of a "<repo>/<chart>" reference, for error
// messages that suggest a follow-up command.
func chartNameOf(ref string) string {
	if _, name, ok := cutLast(ref, "/"); ok {
		return name
	}
	return ref
}

func cutLast(s, sep string) (before, after string, found bool) {
	for i := len(s) - len(sep); i >= 0; i-- {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return s, "", false
}
