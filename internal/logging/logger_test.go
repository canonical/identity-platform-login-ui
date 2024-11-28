package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDebugLogger(t *testing.T) {
	assert := assert.New(t)
	assert.NotPanics(func() { NewLogger("DEBUG") }, "No panic should have been thrown")
}

func TestInvalidLevel(t *testing.T) {
	assert := assert.New(t)
	assert.NotPanics(func() { NewLogger("invalid") }, "No panic should have been thrown")
}
