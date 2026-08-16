// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Eldara-Tech/swarmcli/docker"
)

// flags holds the parsed flag values shared across charts subcommands. Not
// every subcommand reads every field.
type flags struct {
	values       []string // -f/--values, repeatable
	sets         []string // --set, repeatable
	setFiles     []string // --set-file key=path, repeatable
	version      string   // --version, chart version selector
	dryRun       bool
	wait         bool
	requirements bool          // --requirements (template): emit rendered requirements.yaml
	purge        bool          // --purge-volumes
	install      bool          // --install (upgrade)
	reuseValues  bool          // --reuse-values (upgrade)
	diff         bool          // --diff (apply): show each changed release's manifest diff
	revision     int           // --revision (get)
	timeout      time.Duration // --timeout
	historyMax   int           // --history-max
	// skipCompatCheck (--skip-compat-check) downgrades a chart's unmet
	// swarmcliVersion requirement from a refusal to a warning.
	skipCompatCheck bool
	// noRepoUpdate (--no-repo-update) leaves the cached repository indexes
	// alone, so resolution answers out of them and touches no network.
	noRepoUpdate bool
	// forVersion (--for-version) lints a chart against a chart-engine version
	// other than this build's.
	forVersion string
	// resolveImage (--resolve-image) selects the daemon's tag-to-digest
	// resolution at deploy time: always | changed | never.
	resolveImage string

	// seen records the canonical long name of every flag actually passed. The
	// values above cannot answer "was this flag given?" for a flag whose value
	// happens to equal its zero value, and the allow-list check has to reject a
	// flag the operator passed rather than one that merely looks set.
	seen map[string]bool
}

// canonicalFlag maps a flag's short form to the long one the command table
// lists. Long forms are the table's vocabulary, so a row never has to spell a
// flag twice.
var canonicalFlag = map[string]string{"-f": "--values"}

// reject reports the first flag c does not honour. The charts flag set is
// parsed globally — every subcommand understands every flag — so without this
// a valid-but-irrelevant flag is accepted and silently dropped, which reads to
// an operator as "it worked".
func (f flags) reject(c chartsCmd) error {
	allowed := make(map[string]bool, len(c.Flags))
	for _, n := range c.Flags {
		allowed[n] = true
	}
	// Sorted, so a command given two rejected flags always names the same one.
	names := make([]string, 0, len(f.seen))
	for n := range f.seen {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if allowed[n] {
			continue
		}
		if c.FlagHint != "" {
			return fmt.Errorf("charts %s does not accept '%s': %s", c.Name, n, c.FlagHint)
		}
		return fmt.Errorf("charts %s does not accept '%s'", c.Name, n)
	}
	return nil
}

// parse is the prelude every charts subcommand shares: split the arguments,
// reject a flag this command does not honour, and turn either failure into an
// exit code. A returned code >= 0 is the command's result.
func parse(c chartsCmd, args []string) ([]string, flags, int) {
	pos, f, err := parseArgs(args)
	if err != nil {
		return nil, f, usageErr(err.Error())
	}
	if err := f.reject(c); err != nil {
		return nil, f, usageErr(err.Error())
	}
	return pos, f, -1
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
		if f.seen == nil {
			f.seen = map[string]bool{}
		}
		canonical := name
		if long, ok := canonicalFlag[name]; ok {
			canonical = long
		}
		f.seen[canonical] = true
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
		case "--set-file":
			v, err := next()
			if err != nil {
				return nil, f, err
			}
			f.setFiles = append(f.setFiles, v)
		case "--resolve-image":
			v, err := next()
			if err != nil {
				return nil, f, err
			}
			if !docker.ResolveImage(v).Valid() || v == "" {
				return nil, f, fmt.Errorf("invalid value for --resolve-image: '%s' (want always, changed or never)", v)
			}
			f.resolveImage = v
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
				return nil, f, fmt.Errorf("invalid --timeout '%s': %w", v, err)
			}
			f.timeout = d
		case "--history-max":
			v, err := next()
			if err != nil {
				return nil, f, err
			}
			n, err := parseInt(v)
			if err != nil {
				return nil, f, fmt.Errorf("invalid --history-max '%s': %w", v, err)
			}
			f.historyMax = n
		case "--revision":
			v, err := next()
			if err != nil {
				return nil, f, err
			}
			n, err := parseInt(v)
			if err != nil {
				return nil, f, fmt.Errorf("invalid --revision '%s': %w", v, err)
			}
			f.revision = n
		case "--reuse-values":
			f.reuseValues = true
		case "--diff":
			f.diff = true
		case "--dry-run":
			f.dryRun = true
		case "--wait":
			f.wait = true
		case "--requirements":
			f.requirements = true
		case "--purge-volumes":
			f.purge = true
		case "--install":
			f.install = true
		case "--skip-compat-check":
			f.skipCompatCheck = true
		case "--no-repo-update":
			f.noRepoUpdate = true
		case "--for-version":
			v, err := next()
			if err != nil {
				return nil, f, err
			}
			f.forVersion = v
		default:
			return nil, f, fmt.Errorf("unknown flag '%s'", name)
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
		return 0, fmt.Errorf("not a non-negative integer: '%s'", s)
	}
	return n, nil
}
