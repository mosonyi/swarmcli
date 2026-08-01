// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"fmt"
	"iter"
	"maps"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// chartFilesRule is the sentence every refusal below carries.
//
// Refusing an absolute path breaks a chart shape that works today, and there is
// no compatibility flag and no deprecation window, so the error text is the
// whole of the migration path. One that only said "refused" would turn a
// breaking change into a support question.
const chartFilesRule = "a chart may only read files it ships in " + filesDir + "/"

// ResolveManifestFiles returns the chart files a rendered manifest names, and
// refuses every path that cannot mean a file in the chart.
//
// Exactly three compose keys read a path — a config's file:, a secret's file:
// and a service's env_file: — and the docker CLI resolves all three against the
// directory of the compose file it was handed, then reads them itself,
// client-side and as the invoking user, before anything reaches the daemon. In
// docker/cli v28.5.1 that is absPath in cli/compose/loader/loader.go for all
// three, then fileObjectConfig in cli/compose/convert/compose.go for the two
// file: keys and parseEnvFile in cli/compose/loader/envfile.go for env_file:.
//
// swarmcli writes that compose file to a temp directory. So an unguarded
// relative path has always meant "a file in the system temp directory" — which
// any local user can plant, against a temp name predictable enough to know when
// to try — and an absolute one has always meant itself, read as the operator
// into a swarm config that anyone with Docker access can read.
//
// This therefore refuses on the way past, in order: a path that is absolute,
// one that escapes the chart, and one the chart does not contain — the last
// including anything outside files/. None of it depends on where the chart was
// loaded from. Trusting a local-path chart with an absolute path would grant the
// most privilege to vendoring a repository chart to disk, which is the workflow
// that most obscures a chart's origin.
//
// What survives comes back keyed exactly as Chart.Files keys it, and is
// materialised beside the manifest at that same relative path — which is what
// makes file: mean the chart. Only the referenced subset is returned: it is
// persisted in the release record so a rollback replays the bytes the original
// deploy sent, and that record has a hard size ceiling it shares with the
// manifest.
//
// A manifest naming no file at all yields a nil map and no error, which is every
// manifest a chart without a files/ directory can produce.
func ResolveManifestFiles(manifest string, files map[string][]byte) (map[string][]byte, error) {
	refs, err := manifestFileRefs(manifest)
	if err != nil {
		return nil, err
	}
	var out map[string][]byte
	for _, ref := range refs {
		key, err := ref.resolve(files)
		if err != nil {
			return nil, err
		}
		if out == nil {
			out = map[string][]byte{}
		}
		out[key] = files[key]
	}
	return out, nil
}

// fileRef is one path a rendered manifest asks the docker CLI to read, carrying
// the compose key that asked for it so a refusal can name both.
type fileRef struct {
	key  string // "configs.site.file", "services.web.env_file"
	path string // the path exactly as the manifest wrote it
}

// resolve returns the Chart.Files key this reference names, or the error that
// refuses it. Every message names the offending path and states the rule; the
// absolute one additionally names its replacement, because it is the only
// refusal with a working chart behind it.
func (r fileRef) resolve(files map[string][]byte) (string, error) {
	if path.IsAbs(r.path) {
		return "", fmt.Errorf(
			"%s: %q is an absolute path, and %s — copy it into the chart's %s/ and reference it by that path, "+
				"or keep it operator-managed by creating it outside the chart "+
				"(docker config create <name> <path>, or docker secret create) and referencing it with external: true",
			r.key, r.path, chartFilesRule, filesDir)
	}
	// The value came from YAML, which is slash-separated whatever the host is,
	// so this is path and not filepath. Clean resolves every interior "..", so a
	// leading one afterwards is the complete test for leaving the chart root.
	clean := path.Clean(r.path)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%s: %q escapes the chart, and %s", r.key, r.path, chartFilesRule)
	}
	if !strings.HasPrefix(clean, filesDir+"/") {
		return "", fmt.Errorf("%s: %q is outside %s/, and %s — reference it as %q",
			r.key, r.path, filesDir, chartFilesRule, filesDir+"/"+clean)
	}
	if _, ok := files[clean]; !ok {
		return "", fmt.Errorf("%s: %q is not in the chart, and %s", r.key, r.path, chartFilesRule)
	}
	return clean, nil
}

// manifestFileRefs returns every path the manifest asks the docker CLI to read,
// in a stable order so a manifest with more than one bad path always refuses on
// the same one.
//
// The manifest is walked as yaml.Nodes rather than decoded into map[string]any
// because a single non-string key anywhere in a mapping makes yaml.v3 hand back
// a map[any]any for the WHOLE mapping — so one "services: {1: …}" entry would
// make every sibling service invisible to a type assertion, and a guard that
// quietly stops looking is worse than no guard. A node map coerces the key to a
// string, and decoding each entry on its own means one malformed entry cannot
// hide the others.
//
// Anything that is still not the shape compose describes names no path here and
// is left to the deploy, which validates the manifest against the real schema.
func manifestFileRefs(manifest string) ([]fileRef, error) {
	var top map[string]yaml.Node
	if err := yaml.Unmarshal([]byte(manifest), &top); err != nil {
		// A refusal rather than an empty result: this is the guard, and a
		// manifest it cannot parse is one whose file: keys it cannot see.
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	var refs []fileRef
	for _, section := range []string{"configs", "secrets"} {
		for name, node := range entries(top[section]) {
			var entry struct {
				File string `yaml:"file"`
			}
			// An entry with no file: — external: true, or driver-backed — is
			// indistinguishable from file: "" here, and neither names anything.
			if err := node.Decode(&entry); err != nil || entry.File == "" {
				continue
			}
			refs = append(refs, fileRef{key: section + "." + name + ".file", path: entry.File})
		}
	}

	for name, node := range entries(top["services"]) {
		var entry struct {
			EnvFile yaml.Node `yaml:"env_file"`
		}
		if err := node.Decode(&entry); err != nil {
			continue
		}
		// env_file is the one of the three compose gives two shapes: a bare
		// string, or a list of them. Both are read, and the list is walked item
		// by item so an item of the wrong type does not lose the rest.
		items := []*yaml.Node{&entry.EnvFile}
		if entry.EnvFile.Kind == yaml.SequenceNode {
			items = entry.EnvFile.Content
		}
		for _, item := range items {
			var p string
			if err := item.Decode(&p); err != nil || p == "" {
				continue
			}
			refs = append(refs, fileRef{key: "services." + name + ".env_file", path: p})
		}
	}
	return refs, nil
}

// entries yields the named entries of one top-level compose block, sorted by
// name. A block that is absent, or present but not a mapping, yields nothing.
func entries(node yaml.Node) iter.Seq2[string, yaml.Node] {
	var block map[string]yaml.Node
	_ = node.Decode(&block)
	names := slices.Sorted(maps.Keys(block))
	return func(yield func(string, yaml.Node) bool) {
		for _, name := range names {
			if !yield(name, block[name]) {
				return
			}
		}
	}
}
