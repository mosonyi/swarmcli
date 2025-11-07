package swarmlog

import (
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger *zap.SugaredLogger

// Init initializes the global logger.
// It automatically determines the environment using the SWARMCLI_ENV variable:
//
//	SWARMCLI_ENV=dev   → human-readable logs in ~/.local/state/<app>/app-debug.log
//	SWARMCLI_ENV=prod  → JSON logs in in ~/.local/state/<app>/app.log
//
// If unset, defaults to "prod" for safety.
func Init(appName string) {
	mode := detectMode()
	logPath := selectLogPath(appName, mode)

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

	level := detectLogLevel()

	var encoder zapcore.Encoder
	if mode == "dev" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
		zapcore.NewCore(encoder, writer, level)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
		zapcore.NewCore(encoder, writer, level)
	}

	core := zapcore.NewCore(encoder, writer, zap.DebugLevel)
	logger := zap.New(core, zap.AddCaller())
	Logger = logger.Sugar()

	Logger.Infof("Logger initialized in %s mode. Writing to %s", mode, logPath)
}

// Sync flushes pending log entries.
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// InitTest creates a test logger that logs to stdout.
func InitTest() {
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{"stdout"}
	logger, _ := cfg.Build()
	Logger = logger.Sugar()
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
	var fileName string

	if mode == "dev" {
		fileName = "app-debug.log"
	} else {
		fileName = "app.log"

	}

	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		_ = os.MkdirAll(filepath.Join(xdg, appName), 0755)
		return filepath.Join(xdg, appName, fileName)
	}
	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, ".local", "state", appName)
		_ = os.MkdirAll(path, 0755)
		return filepath.Join(path, fileName)
	}

	return filepath.Join("tmp", appName, fileName)
}

func detectLogLevel() zapcore.Level {
	level := strings.ToLower(os.Getenv("LOG_LEVEL"))
	switch level {
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
		// Default: more verbose in dev, quieter in prod
		if detectMode() == "dev" {
			return zap.DebugLevel
		}
		return zap.InfoLevel
	}
}
