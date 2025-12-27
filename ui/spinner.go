package ui

import (
	"github.com/briandowns/spinner"
)

// DefaultSpinnerCharsetIndex is the charset index used across views.
const DefaultSpinnerCharsetIndex = 14

// SpinnerCharAt returns the spinner character for the given frame index.
// Falls back to an ellipsis if spinner charset is not available.
func SpinnerCharAt(frame int) string {
	frames := spinner.CharSets[DefaultSpinnerCharsetIndex]
	if len(frames) == 0 {
		return "…"
	}
	return string(frames[frame%len(frames)])
}

// SpinnerMarker returns the first spinner character (useful as a marker).
func SpinnerMarker() string {
	frames := spinner.CharSets[DefaultSpinnerCharsetIndex]
	if len(frames) == 0 {
		return "…"
	}
	return string(frames[0])
}
