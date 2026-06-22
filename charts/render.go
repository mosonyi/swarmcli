// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"

	"swarmcli/docker"
)

// RenderContext is the data exposed to chart templates, mirroring Helm's
// top-level objects: .Values, .Release, .Chart.
type RenderContext struct {
	Values  map[string]any
	Release ReleaseMeta
	Chart   ChartMeta
}

// ReleaseMeta is the .Release object available to templates.
type ReleaseMeta struct {
	Name      string
	Namespace string
	Revision  int
}

// ChartMeta is the .Chart object available to templates.
type ChartMeta struct {
	Name       string
	Version    string
	AppVersion string
}

// Render evaluates every templates/*.yaml file with text/template + Sprig,
// deep-merges the resulting Compose fragments into a single document, and
// returns it as validated YAML. Files whose names start with "_" (e.g.
// templates/_helpers.tpl) define named templates only and emit no document.
func Render(ch *Chart, ctx RenderContext) (string, error) {
	merged := map[string]any{}

	for _, name := range sortedTemplateNames(ch.Templates) {
		rendered, err := renderOne(ch, name, ctx)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(baseName(name), "_") {
			continue // helper-only file
		}
		if strings.TrimSpace(rendered) == "" {
			continue // empty after rendering (e.g. fully gated by {{ if }})
		}
		frag := map[string]any{}
		if err := yaml.Unmarshal([]byte(rendered), &frag); err != nil {
			return "", fmt.Errorf("template %s produced invalid YAML: %w", name, err)
		}
		merged = deepMerge(merged, frag)
	}

	// The top-level compose `version` key is obsolete for `docker stack deploy`
	// (the engine ignores it) and, because the merged manifest is re-encoded from
	// a map with sorted keys, it would otherwise surface as a confusing trailing
	// line. Drop it so the rendered stack is clean.
	delete(merged, "version")

	if len(merged) == 0 {
		return "", fmt.Errorf("chart %q rendered an empty manifest", ch.Metadata.Name)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(merged); err != nil {
		return "", fmt.Errorf("encode rendered manifest: %w", err)
	}
	_ = enc.Close()

	out := buf.String()
	if err := docker.ValidateStackYAML(out); err != nil {
		return "", fmt.Errorf("rendered manifest is not a valid Docker stack: %w", err)
	}
	return out, nil
}

// renderOne parses every template in the chart (so cross-file named templates
// and {{ template }} / {{ include }} references resolve) and executes the named
// one against ctx.
func renderOne(ch *Chart, name string, ctx RenderContext) (string, error) {
	tmpl := template.New(name).Funcs(sprig.TxtFuncMap()).Funcs(extraFuncs())
	// Parse all templates into the set so helpers are available.
	for _, n := range sortedTemplateNames(ch.Templates) {
		if _, err := tmpl.New(n).Parse(ch.Templates[n]); err != nil {
			return "", fmt.Errorf("template %s: parse error: %w", n, err)
		}
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, ctx); err != nil {
		return "", fmt.Errorf("template %s: %w", name, err)
	}
	return buf.String(), nil
}

// extraFuncs adds Helm-style helpers that Sprig does not provide. "include"
// renders a named template to a string so it can be piped (e.g. into indent).
func extraFuncs() template.FuncMap {
	return template.FuncMap{
		"toYaml": func(v any) (string, error) {
			var b bytes.Buffer
			enc := yaml.NewEncoder(&b)
			enc.SetIndent(2)
			if err := enc.Encode(v); err != nil {
				return "", err
			}
			_ = enc.Close()
			return strings.TrimRight(b.String(), "\n"), nil
		},
	}
}

func sortedTemplateNames(m map[string]string) []string {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func baseName(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}
