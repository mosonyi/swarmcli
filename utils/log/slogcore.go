// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package swarmlog

import (
	"context"
	"log/slog"
	"slices"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitSlog installs a logger that forwards everything to h instead of writing
// a file, so that a program built on log/slog gets this package's output in
// its own format, on its own stream, at its own level.
//
// It exists for consumers that import swarmcli as a library rather than
// running the TUI. Init is wrong for them twice over: it opens a lumberjack
// file under ~/.local/state, which in a container is a file nobody will ever
// read, and it would put a second format on a stream the host program already
// owns. Not calling it at all is worse — L() then returns the no-op logger and
// everything this package's callers report is silently discarded. That was
// swarmcli-cd#72.
//
// The level is h's. LOG_LEVEL and SetLevel do not apply, because the host
// program's handler already decides what it keeps and having two answers to
// that question is how logs go missing.
func InitSlog(h slog.Handler) {
	if raw != nil {
		_ = raw.Sync()
	}
	raw = zap.New(&slogCore{handler: h}, zap.AddCaller())
	logger = &SwarmLogger{raw.Sugar()}
}

// slogCore is a zapcore.Core that hands every entry to an slog.Handler.
type slogCore struct {
	handler slog.Handler
	// attrs are what With accumulated, e.g. the ("docker", "client") pair
	// docker.l() adds to every line it writes.
	attrs []slog.Attr
}

// Enabled implements zapcore.Core. The handler is the only authority on what
// is worth writing.
func (c *slogCore) Enabled(l zapcore.Level) bool {
	return c.handler.Enabled(context.Background(), slogLevel(l))
}

// With implements zapcore.Core.
func (c *slogCore) With(fields []zapcore.Field) zapcore.Core {
	// Clip so two loggers derived from the same parent cannot append into each
	// other's backing array.
	return &slogCore{
		handler: c.handler,
		attrs:   append(slices.Clip(c.attrs), attrs(fields)...),
	}
}

// Check implements zapcore.Core.
func (c *slogCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(e.Level) {
		return ce.AddCore(e, c)
	}
	return ce
}

// Write implements zapcore.Core.
func (c *slogCore) Write(e zapcore.Entry, fields []zapcore.Field) error {
	// A zero PC rather than the entry's caller: slog resolves source from the
	// PC of its own call site, which for a forwarded entry is this function.
	// The caller travels as an attribute instead, below.
	r := slog.NewRecord(e.Time, slogLevel(e.Level), e.Message, 0)
	r.AddAttrs(c.attrs...)
	r.AddAttrs(attrs(fields)...)
	if e.LoggerName != "" {
		r.AddAttrs(slog.String("logger", e.LoggerName))
	}
	if e.Caller.Defined {
		r.AddAttrs(slog.String("caller", e.Caller.TrimmedPath()))
	}
	if e.Stack != "" {
		r.AddAttrs(slog.String("stack", e.Stack))
	}
	return c.handler.Handle(context.Background(), r)
}

// Sync implements zapcore.Core. Nothing is buffered here; whether the
// destination needs flushing is the host program's business.
func (c *slogCore) Sync() error { return nil }

// attrs converts zap fields to slog attributes. It goes through zap's own map
// encoder rather than switching on Field.Type, so every field constructor zap
// has — including the ones it grows later — is handled by construction.
func attrs(fields []zapcore.Field) []slog.Attr {
	if len(fields) == 0 {
		return nil
	}
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(enc)
	}
	out := make([]slog.Attr, 0, len(enc.Fields))
	for k, v := range enc.Fields {
		out = append(out, slog.Any(k, v))
	}
	// Map iteration order is random, and a log line whose fields shuffle
	// between writes is one nothing can diff or test.
	slices.SortFunc(out, func(a, b slog.Attr) int {
		switch {
		case a.Key < b.Key:
			return -1
		case a.Key > b.Key:
			return 1
		}
		return 0
	})
	return out
}

// slogLevel maps a zap level to the nearest slog one. zap's three levels above
// Error all describe a program that is about to stop, which slog spells Error.
func slogLevel(l zapcore.Level) slog.Level {
	switch l {
	case zapcore.DebugLevel:
		return slog.LevelDebug
	case zapcore.InfoLevel:
		return slog.LevelInfo
	case zapcore.WarnLevel:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}
