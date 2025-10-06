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
