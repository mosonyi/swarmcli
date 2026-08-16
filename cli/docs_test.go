// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// updateDocs rewrites the generated blocks instead of asserting on them:
//
//	go test ./cli -run TestGeneratedCommandBlocks -update
//
// It is how a new row reaches the READMEs. Without it this test only reads.
var updateDocs = flag.Bool("update", false, "rewrite the generated README command blocks")

const (
	blockBegin = "<!-- BEGIN generated: charts commands -->"
	blockEnd   = "<!-- END generated -->"
)

// generatedDocs are the files carrying a generated command block. They are the
// two places the command list used to be maintained by hand, and drifted.
var generatedDocs = []string{"../README.md", "../charts/README.md"}

// The command list in README.md and charts/README.md is rendered from
// chartsCommands. Editing either by hand, or adding a row without
// regenerating, fails here — which is the whole point: before this, three
// copies of the list drifted and nothing noticed.
func TestGeneratedCommandBlocks(t *testing.T) {
	want := "```bash\n" + renderCommandsShell() + "```"

	for _, path := range generatedDocs {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			doc := string(raw)

			before, after := split(t, path, doc)
			if *updateDocs {
				require.NoError(t, os.WriteFile(path, []byte(before+want+after), 0o644)) //nolint:gosec // documentation, tracked in git
				return
			}

			got := strings.TrimSuffix(strings.TrimPrefix(doc, before), after)
			require.Equal(t, want, got,
				"%s is stale — regenerate with: go test ./cli -run TestGeneratedCommandBlocks -update", path)
		})
	}
}

// split returns everything up to and including the begin marker, and everything
// from the end marker on, so the block between them can be compared or replaced.
func split(t *testing.T, path, doc string) (before, after string) {
	t.Helper()
	i := strings.Index(doc, blockBegin)
	require.GreaterOrEqual(t, i, 0, "%s has no %s marker", path, blockBegin)
	j := strings.Index(doc, blockEnd)
	require.Greater(t, j, i, "%s has no %s marker after the begin marker", path, blockEnd)
	return doc[:i+len(blockBegin)] + "\n", "\n" + doc[j:]
}
