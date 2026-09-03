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

func TestRequestLoggerWriteParsesSeverity(t *testing.T) {
	logger := NewRequestLogger()

	payload := []byte(`{"time":"2026-08-28T00:00:00Z","severity":"info","message":"hello"}`)
	n, err := logger.Write(payload)
	assert.NoError(t, err)
	assert.Equal(t, len(payload), n)
	assert.Equal(t, []string{"info:hello"}, logger.Get())

	// A payload using the old "level" key yields no severity prefix, which is
	// what makes this the counterpart of setupLogging's FieldMap.
	logger.Clear()
	_, err = logger.Write([]byte(`{"level":"info","message":"hello"}`))
	assert.NoError(t, err)
	assert.Equal(t, []string{":hello"}, logger.Get())
}
