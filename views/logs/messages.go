// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package logsview

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

type FullscreenToggledMsg struct{}

type NodeFilterToggledMsg struct{}
