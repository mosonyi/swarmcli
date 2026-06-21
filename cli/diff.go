// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import "strings"

// lineDiff returns a compact line-oriented diff of a vs b using a longest
// common subsequence. Removed lines are prefixed "-", added lines "+", and
// unchanged lines " ". It is intended for human-readable upgrade previews, not
// machine patching, so it omits hunk headers.
func lineDiff(a, b string) string {
	al := splitLines(a)
	bl := splitLines(b)

	// LCS length table.
	lcs := make([][]int, len(al)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(bl)+1)
	}
	for i := len(al) - 1; i >= 0; i-- {
		for j := len(bl) - 1; j >= 0; j-- {
			if al[i] == bl[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var sb strings.Builder
	i, j := 0, 0
	for i < len(al) && j < len(bl) {
		switch {
		case al[i] == bl[j]:
			sb.WriteString("  " + al[i] + "\n")
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			sb.WriteString("- " + al[i] + "\n")
			i++
		default:
			sb.WriteString("+ " + bl[j] + "\n")
			j++
		}
	}
	for ; i < len(al); i++ {
		sb.WriteString("- " + al[i] + "\n")
	}
	for ; j < len(bl); j++ {
		sb.WriteString("+ " + bl[j] + "\n")
	}
	return sb.String()
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
