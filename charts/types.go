// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

// Package charts implements a Helm-inspired package manager for Docker Swarm.
//
// A chart is a versioned package of Docker Stack (Compose) templates plus
// default values and metadata. Installing a chart produces a release: a Docker
// stack whose revision history is stored append-only in Docker Configs. The
// package is pure Go (no Bubble Tea / TUI) so it can back both the
// non-interactive CLI and, later, a TUI browser view.
package charts

// Label keys applied to every release-history Docker Config. They mirror the
// scheme documented in issue #413 and let list/status/history queries filter
// Configs by release without an external database.
const (
	LabelType         = "com.swarmcli.type"          // always "release"
	LabelRelease      = "com.swarmcli.release"       // release name
	LabelChart        = "com.swarmcli.chart"         // chart name
	LabelChartVersion = "com.swarmcli.chart.version" // chart SemVer
	LabelRevision     = "com.swarmcli.revision"      // revision number
	LabelStatus       = "com.swarmcli.status"        // see Status* constants
	LabelCreated      = "com.swarmcli.created"       // RFC3339 timestamp

	TypeRelease = "release"
)

// Release status values, following Helm's deploy/superseded/failed lifecycle.
const (
	StatusPendingInstall = "pending-install"
	StatusDeployed       = "deployed"
	StatusSuperseded     = "superseded"
	StatusFailed         = "failed"
	StatusUninstalled    = "uninstalled"
)

// Chartfile is the parsed Chart.yaml metadata.
type Chartfile struct {
	APIVersion  string       `yaml:"apiVersion"`
	Name        string       `yaml:"name"`
	Version     string       `yaml:"version"`
	AppVersion  string       `yaml:"appVersion,omitempty"`
	Description string       `yaml:"description,omitempty"`
	Maintainers []Maintainer `yaml:"maintainers,omitempty"`
	// Dependencies are parsed but not resolved in Phase 1 (subcharts are Phase 3).
	Dependencies []Dependency `yaml:"dependencies,omitempty"`
}

// Maintainer identifies a chart maintainer.
type Maintainer struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email,omitempty"`
	URL   string `yaml:"url,omitempty"`
}

// Dependency is a declared subchart requirement (resolution deferred to Phase 3).
type Dependency struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository,omitempty"`
}

// Chart is a loaded chart: its metadata, default values, optional values
// schema, and raw template sources keyed by their path under templates/.
type Chart struct {
	Metadata     Chartfile
	Values       map[string]any    // parsed values.yaml (defaults)
	ValuesRaw    []byte            // raw values.yaml bytes, nil if absent (preserves comments/order)
	Schema       []byte            // raw values.schema.json, nil if absent
	Templates    map[string]string // template path -> source, e.g. "templates/stack.yaml"
	Readme       string            // README.md, empty if absent
	Requirements *Requirements     // parsed requirements.yaml, nil if absent
}

// Requirements is the parsed, defaulted requirements.yaml: the external
// networks/secrets/configs a chart needs. It is optional — a chart without it
// falls back to manifest-driven pre-flight. When present it is authoritative:
// every external resource the rendered manifest references must be declared.
type Requirements struct {
	Networks []NetworkRequirement  `yaml:"networks"`
	Secrets  []ResourceRequirement `yaml:"secrets"`
	Configs  []ResourceRequirement `yaml:"configs"`
}

// NetworkRequirement declares one external network a chart needs. AutoCreate and
// Attachable are pointers so an omitted key defaults to true (preserving the
// historical auto-create-as-attachable-overlay behaviour) while an explicit
// false is distinguishable. After parseRequirements they are always non-nil and
// Driver is non-empty.
type NetworkRequirement struct {
	Name        string `yaml:"name"`
	Driver      string `yaml:"driver"`     // default "overlay"
	Attachable  *bool  `yaml:"attachable"` // default true
	AutoCreate  *bool  `yaml:"autoCreate"` // default true; false => validate-only
	Description string `yaml:"description"`
}

// ResourceRequirement declares one external secret or config a chart needs.
// These cannot be auto-created; Description enriches the remediation message.
type ResourceRequirement struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// RepoEntry is a configured chart repository (name -> index URL).
type RepoEntry struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Index is the parsed index.yaml of a chart repository, mapping a chart name to
// its available versions (Helm repository index format, subset).
type Index struct {
	APIVersion string                  `yaml:"apiVersion"`
	Entries    map[string][]IndexEntry `yaml:"entries"`
}

// IndexEntry describes one published chart version within an Index.
type IndexEntry struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	AppVersion  string   `yaml:"appVersion,omitempty"`
	Description string   `yaml:"description,omitempty"`
	URLs        []string `yaml:"urls"` // tarball download URLs (absolute or index-relative)
	Digest      string   `yaml:"digest,omitempty"`
}

// Release is the payload stored (gzipped) inside a release-history Config. It
// fully describes one deployed revision so it can be inspected or rolled back.
type Release struct {
	Name      string         `yaml:"release"`
	Revision  int            `yaml:"revision"`
	Status    string         `yaml:"status"`
	Chart     ReleaseChart   `yaml:"chart"`
	Values    map[string]any `yaml:"values"`
	Manifest  string         `yaml:"manifest"` // rendered Compose document
	Created   string         `yaml:"created"`  // RFC3339
	Namespace string         `yaml:"namespace"`
	// ManagedNetworks are the external networks swarmcli auto-created for this
	// revision. Persisted so uninstall can report what it left behind (it does
	// not remove them — they may be shared). Omitted for revisions that created
	// none and for records written before this field existed.
	ManagedNetworks []string `yaml:"managedNetworks,omitempty"`
}

// ReleaseChart is the chart reference recorded in a Release.
type ReleaseChart struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion,omitempty"`
}
