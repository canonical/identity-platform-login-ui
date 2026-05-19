// Copyright 2026 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"testing"
)

func TestDebugLogger(t *testing.T) {
	func() {
		_ = recover()
		NewLogger("DEBUG")
	}()
}

func TestInvalidLevel(t *testing.T) {
	func() {
		_ = recover()
		NewLogger("invalid")
	}()
}
