// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package editor

import (
	"os"
	"runtime"
)

// Sense returns the preferred text editor for the current system.
// It checks the EDITOR env var, falls back to VISUAL, and finally
// defaults to notepad on Windows or nano on other platforms.
func Sense() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "nano"
		}
	}
	return editor
}
