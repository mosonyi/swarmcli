package ui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeFrameDimensions_Normal(t *testing.T) {
	spec := ComputeFrameDimensions(100, 40, 0, 0, "", "")
	require.Equal(t, 104, spec.FrameWidth)  // +4 horizontal padding
	require.Equal(t, 40, spec.FrameHeight)
	require.Equal(t, 38, spec.DesiredContentLines) // 40 - 2 vertical padding
}

func TestComputeFrameDimensions_ZeroViewport(t *testing.T) {
	spec := ComputeFrameDimensions(0, 0, 60, 30, "", "")
	require.Equal(t, 64, spec.FrameWidth)  // fallback 60 + 4
	require.Equal(t, 30, spec.FrameHeight) // fallback
}

func TestComputeFrameDimensions_ZeroBoth(t *testing.T) {
	spec := ComputeFrameDimensions(0, 0, 0, 0, "", "")
	require.Equal(t, 84, spec.FrameWidth)  // 80 default + 4
	require.Equal(t, 20, spec.FrameHeight) // 20 default
}

func TestComputeFrameDimensions_HeaderFooter(t *testing.T) {
	spec := ComputeFrameDimensions(100, 40, 0, 0, "header line 1\nheader line 2", "footer")
	// 40 - 2 vertical padding - 2 header lines - 1 footer line = 35
	require.Equal(t, 35, spec.DesiredContentLines)
}

func TestComputeFrameDimensions_NegativeContent(t *testing.T) {
	// Very small height with large header/footer
	spec := ComputeFrameDimensions(100, 3, 0, 0, "h1\nh2\nh3", "f1\nf2\nf3")
	require.Equal(t, 0, spec.DesiredContentLines, "should clamp to 0")
}
