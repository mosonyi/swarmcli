// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// MergeValues computes the effective values for a render, applying Helm's
// precedence: chart defaults < each --values file (in order) < --set overrides.
// Maps are deep-merged; scalars and sequences replace.
func MergeValues(defaults map[string]any, files [][]byte, sets []string) (map[string]any, error) {
	out := deepCopyMap(defaults)

	for i, raw := range files {
		parsed := map[string]any{}
		if err := yaml.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("values file #%d: %w", i+1, err)
		}
		out = deepMerge(out, parsed)
	}

	for _, s := range sets {
		if err := applySet(out, s); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// applySet applies one or more comma-separated "a.b.c=value" overrides to dst.
// Values are type-inferred (bool, int, float, else string). List indexing is
// intentionally not supported in Phase 1.
func applySet(dst map[string]any, expr string) error {
	for _, pair := range splitSet(expr) {
		eq := strings.IndexByte(pair, '=')
		if eq < 0 {
			return fmt.Errorf("--set %q: expected key=value", pair)
		}
		path := strings.TrimSpace(pair[:eq])
		val := inferScalar(strings.TrimSpace(pair[eq+1:]))
		if path == "" {
			return fmt.Errorf("--set %q: empty key", pair)
		}
		setPath(dst, strings.Split(path, "."), val)
	}
	return nil
}

// splitSet splits on commas that are not escaped with a backslash, so values
// may contain literal commas via "\,".
func splitSet(s string) []string {
	var parts []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == ',' {
			cur.WriteByte(',')
			i++
			continue
		}
		if s[i] == ',' {
			parts = append(parts, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(s[i])
	}
	parts = append(parts, cur.String())
	return parts
}

// setPath assigns val at the dotted key path, creating intermediate maps. An
// intermediate scalar is overwritten with a map so the deeper key can be set.
func setPath(m map[string]any, keys []string, val any) {
	for i := 0; i < len(keys)-1; i++ {
		next, ok := m[keys[i]].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[keys[i]] = next
		}
		m = next
	}
	m[keys[len(keys)-1]] = val
}

// inferScalar converts a --set string into bool, int, float64, or string.
func inferScalar(s string) any {
	switch s {
	case "true":
		return true
	case "false":
		return false
	case "null", "nil":
		return nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return int(i)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

// deepMerge recursively merges src into dst and returns dst. Nested maps merge;
// every other type (scalars, sequences) replaces the destination value.
func deepMerge(dst, src map[string]any) map[string]any {
	for k, sv := range src {
		if sm, ok := sv.(map[string]any); ok {
			if dm, ok := dst[k].(map[string]any); ok {
				dst[k] = deepMerge(dm, sm)
				continue
			}
		}
		dst[k] = sv
	}
	return dst
}

// deepCopyMap clones a values map so merges never mutate a chart's defaults.
func deepCopyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = deepCopyValue(v)
	}
	return out
}

func deepCopyValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return deepCopyMap(t)
	case []any:
		cp := make([]any, len(t))
		for i, e := range t {
			cp[i] = deepCopyValue(e)
		}
		return cp
	default:
		return v
	}
}

// ValidateValues checks merged values against a chart's values.schema.json
// (JSON Schema). A nil/empty schema is a no-op.
func ValidateValues(schema []byte, values map[string]any) error {
	if len(bytes.TrimSpace(schema)) == 0 {
		return nil
	}
	compiler := jsonschema.NewCompiler()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		return fmt.Errorf("values.schema.json is not valid JSON: %w", err)
	}
	if err := compiler.AddResource("values.schema.json", doc); err != nil {
		return fmt.Errorf("values.schema.json: %w", err)
	}
	sch, err := compiler.Compile("values.schema.json")
	if err != nil {
		return fmt.Errorf("values.schema.json: %w", err)
	}
	if err := sch.Validate(normalizeForSchema(values)); err != nil {
		return fmt.Errorf("values failed schema validation: %w", err)
	}
	return nil
}

// normalizeForSchema converts YAML-decoded values (map[string]any) into the
// any-keyed shape the JSON Schema validator expects.
func normalizeForSchema(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[k] = normalizeForSchema(e)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = normalizeForSchema(e)
		}
		return out
	case int:
		return float64(t)
	default:
		return v
	}
}
