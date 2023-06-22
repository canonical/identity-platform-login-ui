package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDebugLogger(t *testing.T) {
	assert := assert.New(t)
	assert.NotPanics(func() { NewLogger("DEBUG", "log.txt") }, "No panic should have been thrown")
}

func TestInvalidLevel(t *testing.T) {
	assert := assert.New(t)
	assert.NotPanics(func() { NewLogger("invalid", "log.txt") }, "No panic should have been thrown")
}
