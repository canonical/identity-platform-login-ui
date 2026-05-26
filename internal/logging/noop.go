// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

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
