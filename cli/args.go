// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// flags holds the parsed flag values shared across charts subcommands. Not
// every subcommand reads every field.
type flags struct {
	values      []string // -f/--values, repeatable
	sets        []string // --set, repeatable
	version     string   // --version, chart version selector
	dryRun      bool
	wait        bool
	debug       bool
	purge       bool          // --purge-volumes
	install     bool          // --install (upgrade)
	reuseValues bool          // --reuse-values (upgrade)
	revision    int           // --revision (get)
	timeout     time.Duration // --timeout
	historyMax  int           // --history-max
}

// parseArgs splits raw args into positionals and flags. It understands the
// long/short forms used by the charts commands and rejects unknown flags so
// typos fail loudly (matching the TUI's strict-flag philosophy).
func parseArgs(args []string) ([]string, flags, error) {
	var pos []string
	f := flags{timeout: 0}

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" { // everything after is positional
			pos = append(pos, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") {
			pos = append(pos, a)
			continue
		}

		name, inlineVal, hasInline := splitFlag(a)
		next := func() (string, error) {
			if hasInline {
				return inlineVal, nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("flag %s requires a value", a)
			}
			i++
			return args[i], nil
		}

		switch name {
		case "-f", "--values":
			v, err := next()
			if err != nil {
				return nil, f, err
			}
			f.values = append(f.values, v)
		case "--set":
			v, err := next()
			if err != nil {
				return nil, f, err
			}
			f.sets = append(f.sets, v)
		case "--version":
			v, err := next()
			if err != nil {
				return nil, f, err
			}
			f.version = v
		case "--timeout":
			v, err := next()
			if err != nil {
				return nil, f, err
			}
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, f, fmt.Errorf("invalid --timeout %q: %w", v, err)
			}
			f.timeout = d
		case "--history-max":
			v, err := next()
			if err != nil {
				return nil, f, err
			}
			n, err := parseInt(v)
			if err != nil {
				return nil, f, fmt.Errorf("invalid --history-max %q: %w", v, err)
			}
			f.historyMax = n
		case "--revision":
			v, err := next()
			if err != nil {
				return nil, f, err
			}
			n, err := parseInt(v)
			if err != nil {
				return nil, f, fmt.Errorf("invalid --revision %q: %w", v, err)
			}
			f.revision = n
		case "--reuse-values":
			f.reuseValues = true
		case "--dry-run":
			f.dryRun = true
		case "--wait":
			f.wait = true
		case "--debug":
			f.debug = true
		case "--purge-volumes":
			f.purge = true
		case "--install":
			f.install = true
		default:
			return nil, f, fmt.Errorf("unknown flag %q", name)
		}
	}
	return pos, f, nil
}

// splitFlag splits "--key=value" into ("--key", "value", true); for a bare
// "--key" it returns ("--key", "", false).
func splitFlag(a string) (name, val string, hasVal bool) {
	if eq := strings.IndexByte(a, '='); eq >= 0 {
		return a[:eq], a[eq+1:], true
	}
	return a, "", false
}

// parseInt parses a non-negative integer, rejecting trailing garbage (unlike
// fmt.Sscanf, which stops at the first non-digit) and negative values, which
// none of the callers (--revision, --history-max) accept.
func parseInt(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return 0, fmt.Errorf("not a non-negative integer: %q", s)
	}
	return n, nil
}
