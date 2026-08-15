// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package configsview

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The grow column must be the last one, and there must be only one.
//
// A grow column in the middle does not remove the void a wide terminal opens,
// it relocates it to just after that column; two of them split the slack and
// open two. Both were shipped and reverted before this rule was written down.
func TestExactlyOneGrowColumnAndItIsLast(t *testing.T) {
	cols := testModel().buildColumns()
	require.NotEmpty(t, cols)

	var growing []string
	for _, c := range cols {
		if c.Grow {
			growing = append(growing, c.Label)
		}
	}
	require.Len(t, growing, 1, "exactly one column may absorb leftover width")
	require.Equal(t, cols[len(cols)-1].Label, growing[0],
		"and it must be the last column, or the gap merely moves")
}
