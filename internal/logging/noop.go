package logging

import (
	"go.uber.org/zap"
)

func NewNoopLogger() *zap.SugaredLogger {
	return zap.NewNop().Sugar()
}
