package ui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDistributeColumns_ExactFit(t *testing.T) {
	// totalWidth=20, 1 gap of 2, cols sum to 18 → exact fit
	result := DistributeColumns(20, 1, 2, []int{8, 10}, []int{0})
	require.Equal(t, []int{8, 10}, result)
}

func TestDistributeColumns_Expand(t *testing.T) {
	// totalWidth=30, 1 gap of 2, cols sum to 18, available=28, extra=10 → first flex gets it
	result := DistributeColumns(30, 1, 2, []int{8, 10}, []int{0})
	require.Equal(t, []int{18, 10}, result)
}

func TestDistributeColumns_Shrink(t *testing.T) {
	// totalWidth=15, 1 gap of 2, available=13, cols sum=18, need to shrink 5
	result := DistributeColumns(15, 1, 2, []int{8, 10}, []int{1})
	sum := 0
	for _, v := range result {
		sum += v
	}
	require.Equal(t, 13, sum, "columns should sum to available width")
}

func TestDistributeColumns_ZeroWidth(t *testing.T) {
	result := DistributeColumns(0, 1, 2, []int{8, 10}, []int{0})
	require.Equal(t, []int{8, 10}, result)
}

func TestDistributeColumns_NegativeAvailable(t *testing.T) {
	// totalWidth=2, 1 gap of 10, available=-8 → all cols get minimum 1
	result := DistributeColumns(2, 1, 10, []int{0, 0}, []int{0})
	for _, v := range result {
		require.GreaterOrEqual(t, v, 1)
	}
}

func TestDistributeColumns_NoFlex(t *testing.T) {
	// No flex indices, extra goes to last column
	result := DistributeColumns(25, 1, 2, []int{8, 10}, nil)
	require.Equal(t, 15, result[1], "extra space should go to last column")
}
