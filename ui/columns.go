// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package ui

// DistributeColumns adjusts column widths to fit `totalWidth` after accounting
// for `gapCount` gaps of `gapWidth` each. `cols` contains preferred/min widths
// for each column. `flexIndices` are indexes into `cols` that are allowed to
// absorb remaining space when total cols < available, or to be reduced when
// cols exceed available. The function returns a new slice with adjusted widths.
func DistributeColumns(totalWidth, gapCount, gapWidth int, cols []int, flexIndices []int) []int {
	if totalWidth <= 0 {
		return cols
	}

	// Copy to avoid mutating caller slice
	adjusted := make([]int, len(cols))
	copy(adjusted, cols)

	totalGaps := gapCount * gapWidth
	available := totalWidth - totalGaps
	if available <= 0 {
		// No space for columns, return minimums
		for i := range adjusted {
			if adjusted[i] <= 0 {
				adjusted[i] = 1
			}
		}
		return adjusted
	}

	sum := 0
	for _, v := range adjusted {
		sum += v
	}

	if sum == available {
		return adjusted
	}

	if sum < available {
		// Add remaining to the first flexible column
		remaining := available - sum
		if len(flexIndices) == 0 {
			// Add to last column if no flex indices specified
			adjusted[len(adjusted)-1] += remaining
			return adjusted
		}
		adjusted[flexIndices[0]] += remaining
		return adjusted
	}

	// sum > available: reduce flexible columns starting from the largest
	// until we fit. Do not reduce below 1.
	// Build list of flex idxs sorted by current size descending
	// Simple selection loop is sufficient for expected small number of columns
	for sum > available && len(flexIndices) > 0 {
		// find index of largest flexible column
		largestIdx := -1
		largestVal := -1
		for _, idx := range flexIndices {
			if adjusted[idx] > largestVal {
				largestVal = adjusted[idx]
				largestIdx = idx
			}
		}
		if largestIdx == -1 || largestVal <= 1 {
			break
		}
		// reduce by 1
		adjusted[largestIdx]--
		sum--
	}

	// If still too big, reduce non-flex columns as last resort
	i := 0
	for sum > available && i < len(adjusted) {
		if adjusted[i] > 1 {
			adjusted[i]--
			sum--
			continue
		}
		i++
	}

	return adjusted
}
