package logging

import (
	"go.uber.org/zap"
)

func NewNoopLogger() *Logger {
	return &Logger{
		SugaredLogger: zap.NewNop().Sugar(),
		security:      &SecurityLogger{l: zap.NewNop()},
	}
}
