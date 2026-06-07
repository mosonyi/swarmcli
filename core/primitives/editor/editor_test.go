// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package editor

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSense_EditorSet(t *testing.T) {
	t.Setenv("EDITOR", "my-custom-editor")
	t.Setenv("VISUAL", "my-visual-editor")

	require.Equal(t, "my-custom-editor", Sense())
}

func TestSense_VisualSet(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "my-visual-editor")

	require.Equal(t, "my-visual-editor", Sense())
}

func TestSense_Default(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")

	expected := "nano"
	if runtime.GOOS == "windows" {
		expected = "notepad"
	}

	require.Equal(t, expected, Sense())
}
