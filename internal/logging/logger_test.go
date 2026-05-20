// Copyright 2026 Canonical Ltd
// SPDX-License-Identifier: AGPL-3.0-only

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
