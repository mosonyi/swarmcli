// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package logsview

import "time"

type InitStreamMsg struct {
	Lines    chan string
	Errs     chan error
	MaxLines int
}

type LineMsg struct {
	Line string
}

type StreamErrMsg struct {
	Err error
}

type StreamDoneMsg struct{}

type WrapToggledMsg struct{}

type NodeFilterToggledMsg struct{}

type HideStoppedToggledMsg struct{}

// MarkInsertedMsg signals that the reader put a separator in the buffer, so the
// viewport has to be rebuilt around it.
type MarkInsertedMsg struct{}

// FadeTickMsg drives the expiry of the new-line highlight: without a beat, a
// line that arrives into silence has nothing to redraw it back to normal.
type FadeTickMsg time.Time

// Highlight timings. All three are vars, not consts, so tests can shrink them:
// a tea.Tick cmd invoked synchronously blocks for its whole interval, and a
// test that waited out a real highlight would sit here for over a second.
var (
	// highlightDuration is how long a freshly arrived line stays bold.
	highlightDuration = 1200 * time.Millisecond
	// fadeTickInterval is how closely the expiry follows highlightDuration.
	fadeTickInterval = 150 * time.Millisecond
	// highlightArmDelay is the gap between arrivals that marks the end of the
	// backlog replay a stream opens with.
	highlightArmDelay = 300 * time.Millisecond
)

// backlogTail is how many lines of history the stream asks Docker to replay.
// The highlight reads it too — it is the upper bound on the lines that are
// history rather than news — so the two cannot drift.
const backlogTail = 200
