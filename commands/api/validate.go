// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package api //nolint:revive // standard short package name

import (
	"fmt"
	"sort"

	"github.com/Eldara-Tech/swarmcli/args"
	"github.com/Eldara-Tech/swarmcli/registry"
)

// validateFlags rejects any flag in parsed that the command's spec does
// not declare. Long-form `--flag` tokens are the only ones that reach
// parsed.Flags (the parser leaves single-dash tokens as positionals),
// so short aliases need no validation here. A close declared flag is
// suggested via edit distance.
func validateFlags(cmdName string, spec registry.CommandSpec, parsed args.Args) error {
	allowed := make(map[string]struct{}, len(spec.Flags)*2)
	for _, f := range spec.Flags {
		allowed[f.Name] = struct{}{}
		if f.Short != "" {
			allowed[f.Short] = struct{}{}
		}
	}

	// Deterministic iteration so the reported flag/suggestion is stable.
	keys := make([]string, 0, len(parsed.Flags))
	for k := range parsed.Flags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if _, ok := allowed[k]; ok {
			continue
		}
		msg := fmt.Sprintf("unknown flag --%s for :%s", k, cmdName)
		if s := suggestFlag(k, spec.Flags); s != "" {
			msg += fmt.Sprintf(", did you mean --%s?", s)
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// suggestFlag returns the closest declared long flag name to input
// within an edit-distance threshold, or "" if none is close enough.
// Ties break alphabetically for deterministic tests.
func suggestFlag(input string, flags []registry.FlagSpec) string {
	threshold := len(input) / 3
	if threshold < 2 {
		threshold = 2
	}

	best := ""
	bestDist := threshold + 1
	for _, f := range flags {
		d := registry.Distance(input, f.Name)
		if d < bestDist || (d == bestDist && f.Name < best) {
			best, bestDist = f.Name, d
		}
	}
	if bestDist > threshold {
		return ""
	}
	return best
}
