package swarmlog

import (
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// Keep the global logger private to prevent uninitialized access.
	logger *SwarmLogger
	raw    *zap.Logger

	// Noop logger as safe fallback when not initialized.
	noopLogger = &SwarmLogger{zap.NewNop().Sugar()}

	// Atomic log level allows dynamic runtime level changes if desired.
	atomicLevel zap.AtomicLevel
)

// SwarmLogger wraps zap’s SugaredLogger for convenience.
type SwarmLogger struct {
	*zap.SugaredLogger
}

// With adds structured fields to the logger and returns a new instance.
func (l *SwarmLogger) With(args ...interface{}) *SwarmLogger {
	if l == nil {
		return noopLogger
	}
	return &SwarmLogger{l.SugaredLogger.With(args...)}
}

// L returns the global logger or a no-op fallback if uninitialized.
func L() *SwarmLogger {
	if logger == nil {
		return noopLogger
	}
	return logger
}

// Init initializes the global logger.
//
// It automatically determines the environment using SWARMCLI_ENV:
//
//   - SWARMCLI_ENV=dev   → human-readable logs in ~/.local/state/<app>/app-debug.log
//   - SWARMCLI_ENV=prod  → JSON logs in ~/.local/state/<app>/app.log
//
// The log level is controlled via LOG_LEVEL (debug, info, warn, error, etc).
// If unset, defaults to debug in dev mode and info in prod mode.
func Init(appName string) {
	mode := detectMode()
	logPath := selectLogPath(appName, mode)

	atomicLevel = zap.NewAtomicLevelAt(detectLogLevel())

	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    20, // MB
		MaxBackups: 5,
		MaxAge:     14, // days
		Compress:   true,
	})

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	var encoder zapcore.Encoder
	if mode == "dev" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	core := zapcore.NewCore(encoder, writer, atomicLevel)
	raw = zap.New(core, zap.AddCaller())
	logger = &SwarmLogger{raw.Sugar()}

	logger.Infof("logger initialized in %s mode. Writing to %s", mode, logPath)
}

// Sync flushes any buffered log entries.
func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}

// InitTest creates a lightweight logger for tests that logs to stdout.
func InitTest() {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	cfg.OutputPaths = []string{"stdout"}
	raw, _ = cfg.Build(zap.AddCaller())
	logger = &SwarmLogger{raw.Sugar()}
}

// SetLevel allows changing the log level at runtime.
func SetLevel(level zapcore.Level) {
	if atomicLevel != (zap.AtomicLevel{}) {
		atomicLevel.SetLevel(level)
	}
}

// detectMode determines dev or prod mode from SWARMCLI_ENV.
func detectMode() string {
	env := strings.ToLower(os.Getenv("SWARMCLI_ENV"))
	switch env {
	case "dev", "development":
		return "dev"
	default:
		return "prod"
	}
}

// selectLogPath picks a standard file location for logs.
func selectLogPath(appName, mode string) string {
	fileName := "app.log"
	if mode == "dev" {
		fileName = "app-debug.log"
	}

	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		path := filepath.Join(xdg, appName)
		_ = os.MkdirAll(path, 0755)
		return filepath.Join(path, fileName)
	}

	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, ".local", "state", appName)
		_ = os.MkdirAll(path, 0755)
		return filepath.Join(path, fileName)
	}

	// Fallback for restrictive environments
	path := filepath.Join(os.TempDir(), appName)
	_ = os.MkdirAll(path, 0755)
	return filepath.Join(path, fileName)
}

// detectLogLevel picks the initial log level from LOG_LEVEL.
func detectLogLevel() zapcore.Level {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		return zap.DebugLevel
	case "warn", "warning":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "dpanic":
		return zap.DPanicLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		if detectMode() == "dev" {
			return zap.DebugLevel
		}
		return zap.InfoLevel
	}
}
