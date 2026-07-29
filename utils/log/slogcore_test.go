// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package swarmlog

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"go.uber.org/zap/zapcore"
)

// bridge installs a slog-backed logger writing text into a buffer, and puts
// the package's previous logger back afterwards.
func bridge(t *testing.T, level slog.Level) *bytes.Buffer {
	t.Helper()
	previous, previousRaw := logger, raw
	t.Cleanup(func() { logger, raw = previous, previousRaw })

	var buf bytes.Buffer
	InitSlog(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level}))
	return &buf
}

func TestInitSlogForwardsEveryLevel(t *testing.T) {
	for _, tc := range []struct {
		write func(*SwarmLogger)
		want  string
	}{
		{func(l *SwarmLogger) { l.Debug("a message") }, "level=DEBUG"},
		{func(l *SwarmLogger) { l.Info("a message") }, "level=INFO"},
		{func(l *SwarmLogger) { l.Warn("a message") }, "level=WARN"},
		{func(l *SwarmLogger) { l.Error("a message") }, "level=ERROR"},
		// zap's levels above Error all mean the program is stopping, which slog
		// spells Error. DPanic is the only one testable without dying.
		{func(l *SwarmLogger) { l.DPanic("a message") }, "level=ERROR"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			buf := bridge(t, slog.LevelDebug)
			tc.write(L())
			got := buf.String()
			if !strings.Contains(got, tc.want) || !strings.Contains(got, "a message") {
				t.Errorf("wrote %q, want %q and the message", got, tc.want)
			}
		})
	}
}

// The formatted variants are what this package's callers actually use.
func TestInitSlogForwardsFormattedMessages(t *testing.T) {
	buf := bridge(t, slog.LevelDebug)
	L().Warnf("snapshot: cli.Info failed (%d attempts)", 3)
	if !strings.Contains(buf.String(), "snapshot: cli.Info failed (3 attempts)") {
		t.Errorf("wrote %q, want the formatted message", buf.String())
	}
}

// docker.l() derives a logger with With on every call, so this is the shape
// the real callers take.
func TestWithAttributesReachTheHandler(t *testing.T) {
	buf := bridge(t, slog.LevelDebug)
	L().With("docker", "client").Info("connected")

	got := buf.String()
	if !strings.Contains(got, "docker=client") {
		t.Errorf("wrote %q, want the With attribute", got)
	}
	if !strings.Contains(got, "msg=connected") {
		t.Errorf("wrote %q, want the message", got)
	}
}

// Two loggers derived from one parent must not see each other's attributes.
func TestDerivedLoggersDoNotShareAttributes(t *testing.T) {
	buf := bridge(t, slog.LevelDebug)
	base := L().With("pkg", "docker")
	base.With("view", "stacks").Info("one")
	buf.Reset()
	base.With("view", "nodes").Info("two")

	got := buf.String()
	if strings.Contains(got, "stacks") {
		t.Errorf("wrote %q, want no trace of the sibling logger's attribute", got)
	}
	if !strings.Contains(got, "view=nodes") || !strings.Contains(got, "pkg=docker") {
		t.Errorf("wrote %q, want its own attribute and the parent's", got)
	}
}

// The handler decides what is kept — LOG_LEVEL and SetLevel do not apply once
// the bridge is installed, and a level the handler drops must cost nothing.
func TestTheHandlerLevelFilters(t *testing.T) {
	buf := bridge(t, slog.LevelWarn)
	L().Info("dropped")
	L().Warn("kept")

	got := buf.String()
	if strings.Contains(got, "dropped") {
		t.Errorf("wrote %q, want the info line dropped by the handler", got)
	}
	if !strings.Contains(got, "kept") {
		t.Errorf("wrote %q, want the warn line", got)
	}
}

func TestDroppedLevelsAreNotEnabled(t *testing.T) {
	bridge(t, slog.LevelWarn)
	if L().Desugar().Core().Enabled(zapcore.InfoLevel) {
		t.Error("info is enabled, want the handler's level to disable it")
	}
	if !L().Desugar().Core().Enabled(zapcore.ErrorLevel) {
		t.Error("error is disabled, want the handler's level to allow it")
	}
}

// Typed zap fields go through zap's own encoder, so anything it can encode
// arrives without this package knowing the type.
func TestTypedFieldsAreConverted(t *testing.T) {
	buf := bridge(t, slog.LevelDebug)
	L().Infow("service created", "replicas", 3, "name", "whoami", "ok", true)

	got := buf.String()
	for _, want := range []string{"replicas=3", "name=whoami", "ok=true"} {
		if !strings.Contains(got, want) {
			t.Errorf("wrote %q, want %q", got, want)
		}
	}
}

// Field order must be stable: the map encoder's iteration order is not.
func TestFieldOrderIsStable(t *testing.T) {
	buf := bridge(t, slog.LevelDebug)
	for range 8 {
		L().Infow("m", "zeta", 1, "alpha", 2, "mu", 3)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for _, l := range lines[1:] {
		if fieldsOf(l) != fieldsOf(lines[0]) {
			t.Fatalf("field order varies between writes:\n%s\n%s", lines[0], l)
		}
	}
}

// fieldsOf drops the timestamp, which is the one part that legitimately differs.
func fieldsOf(line string) string {
	_, rest, _ := strings.Cut(line, " ")
	return rest
}

// zap.AddCaller is applied to the bridged logger, and the caller is only
// useful if it survives the hop.
func TestTheCallerIsCarried(t *testing.T) {
	buf := bridge(t, slog.LevelDebug)
	L().Info("a message")
	if !strings.Contains(buf.String(), "caller=log/slogcore_test.go") {
		t.Errorf("wrote %q, want the caller attribute", buf.String())
	}
}

func TestInitSlogReplacesTheGlobalLogger(t *testing.T) {
	buf := bridge(t, slog.LevelDebug)
	// The package-level L(), not the returned one: every caller in this
	// repository goes through it.
	L().Info("through the global")
	if !strings.Contains(buf.String(), "through the global") {
		t.Errorf("wrote %q, want L() to be the bridged logger", buf.String())
	}
}
