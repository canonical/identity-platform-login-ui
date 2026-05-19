// Copyright 2026 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

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
