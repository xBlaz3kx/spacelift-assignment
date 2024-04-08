package observability

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

func init() {
	logger := newLogger("debug")
	zap.ReplaceGlobals(logger)
}

// newLogger creates a new JSON logger with the given log level, writing logs to stdout and stderr.
func newLogger(logLevel string) *zap.Logger {

	level := zapcore.InfoLevel
	switch logLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "warning":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	stdout := zapcore.Lock(os.Stdout)
	stderr := zapcore.Lock(os.Stderr)

	stdoutLevelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= level && l < zapcore.ErrorLevel
	})
	stderrLevelEnabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= level && l >= zapcore.ErrorLevel
	})

	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, stdout, stdoutLevelEnabler),
		zapcore.NewCore(encoder, stderr, stderrLevelEnabler),
	)

	return zap.New(core)
}
