package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestLogger(t *testing.T) {
	logger := NewRequestLogger()
	logger.Push("msg 1")
	logger.Push("msg 2")

	logs := logger.Get()
	assert.Equal(t, logs, []string{
		"msg 1",
		"msg 2",
	})

	logger.Clear()
	logs = logger.Get()
	assert.Equal(t, logs, []string{})
}
