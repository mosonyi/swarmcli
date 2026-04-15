// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package swarmlog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDetectMode_Dev(t *testing.T) {
	t.Setenv("SWARMCLI_ENV", "dev")
	require.Equal(t, "dev", detectMode())
}

func TestDetectMode_Development(t *testing.T) {
	t.Setenv("SWARMCLI_ENV", "development")
	require.Equal(t, "dev", detectMode())
}

func TestDetectMode_Prod(t *testing.T) {
	t.Setenv("SWARMCLI_ENV", "prod")
	require.Equal(t, "prod", detectMode())
}

func TestDetectMode_Empty(t *testing.T) {
	t.Setenv("SWARMCLI_ENV", "")
	require.Equal(t, "prod", detectMode())
}

func TestDetectLogLevel_Debug(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	require.Equal(t, zap.DebugLevel, detectLogLevel())
}

func TestDetectLogLevel_Error(t *testing.T) {
	t.Setenv("LOG_LEVEL", "error")
	require.Equal(t, zap.ErrorLevel, detectLogLevel())
}

func TestDetectLogLevel_DefaultDev(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("SWARMCLI_ENV", "dev")
	require.Equal(t, zap.DebugLevel, detectLogLevel())
}

func TestDetectLogLevel_DefaultProd(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("SWARMCLI_ENV", "prod")
	require.Equal(t, zap.InfoLevel, detectLogLevel())
}

func TestSelectLogPath_XDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)
	path := selectLogPath("testapp", "prod")
	require.Equal(t, filepath.Join(tmp, "testapp", "app.log"), path)
}

func TestSelectLogPath_Home(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	path := selectLogPath("testapp", "prod")
	require.Equal(t, filepath.Join(home, ".local", "state", "testapp", "app.log"), path)
}

func TestSelectLogPath_DevFilename(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)
	path := selectLogPath("testapp", "dev")
	require.Contains(t, path, "app-debug.log")
}

func TestSelectLogPath_ProdFilename(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)
	path := selectLogPath("testapp", "prod")
	require.Contains(t, path, "app.log")
	require.NotContains(t, path, "debug")
}

func TestL_Uninitialized(t *testing.T) {
	old := logger
	logger = nil
	defer func() { logger = old }()

	l := L()
	require.NotNil(t, l, "should return noop logger, not nil")
}

func TestInitTestIfTestLogEnv_NoEnv(t *testing.T) {
	old := logger
	defer func() { logger = old }()

	t.Setenv("TEST_LOG", "")
	InitTestIfTestLogEnv()
	require.NotNil(t, logger, "should set noop logger")
}
