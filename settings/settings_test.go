// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// withHome points the package's home-dir seam at a temp dir for the test and
// restores it afterwards.
func withHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	orig := userHomeDirFn
	userHomeDirFn = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeDirFn = orig })
	return home
}

func TestSaveLoadRoundTrip(t *testing.T) {
	home := withHome(t)

	require.NoError(t, Settings{DismissedUpdateVersion: "v1.9.0"}.Save())

	// File lands at ~/.config/swarmcli/update-notice.json.
	_, err := os.Stat(filepath.Join(home, relPath))
	require.NoError(t, err)

	require.Equal(t, "v1.9.0", Load().DismissedUpdateVersion)
}

func TestLoadMissingFileIsZeroValue(t *testing.T) {
	withHome(t)
	require.Equal(t, Settings{}, Load())
}

func TestLoadCorruptFileIsZeroValue(t *testing.T) {
	home := withHome(t)
	p := filepath.Join(home, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte("{not json"), 0o644))

	require.Equal(t, Settings{}, Load())
}

func TestLoadHonorsHomeDirSeam(t *testing.T) {
	home := withHome(t)
	require.NoError(t, Settings{DismissedUpdateVersion: "v2.0.0"}.Save())

	// A second home with no file yields the zero value, proving Load reads
	// from the seam rather than a fixed path.
	other := t.TempDir()
	userHomeDirFn = func() (string, error) { return other, nil }
	require.Equal(t, Settings{}, Load())

	userHomeDirFn = func() (string, error) { return home, nil }
	require.Equal(t, "v2.0.0", Load().DismissedUpdateVersion)
}
