package logging

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
	security *SecurityLogger
}

func (l *Logger) Security() SecurityLoggerInterface {
	return l.security
}

func (l *Logger) Sync() {
	l.security.Sync()
	l.SugaredLogger.Desugar().Sync()
}

func NewLogger(l string) *Logger {
	logger := new(Logger)
	logger.SugaredLogger = NewServiceLogger(l)
	logger.security = NewSecurityLogger(l)
	return logger
}

func NewServiceLogger(l string) *zap.SugaredLogger {
	var lvl zapcore.Level

	switch strings.ToLower(l) {
	case "debug":
		lvl = zap.DebugLevel
	case "info":
		lvl = zap.InfoLevel
	case "warning":
		lvl = zap.WarnLevel
	case "error":
		lvl = zap.ErrorLevel
	case "critical":
		lvl = zap.DPanicLevel
	}

	c := zapcore.EncoderConfig{
		MessageKey:  "description",
		LevelKey:    "level",
		EncodeLevel: zapcore.CapitalLevelEncoder,
		TimeKey:     "datetime",
		EncodeTime:  zapcore.RFC3339NanoTimeEncoder,
	}

	encoder := zapcore.NewJSONEncoder(c)
	encoder.AddString("type", "service")
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), lvl)

	return zap.New(core).Sugar()
}
