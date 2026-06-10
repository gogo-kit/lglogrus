package loglogrus

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gogo-kit/log"
)

func newLogger(buf *bytes.Buffer, keys log.Keys) log.Logger {
	return log.New(log.Config{
		Service:     "order-service",
		Environment: "test",
		Output:      buf,
		Keys:        keys,
		Driver:      Driver,
	})
}

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	require.Equal(t, 1, strings.Count(buf.String(), "\n"), "event must be a single line")
	var m map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	return m
}

func TestLogrusInfoContract(t *testing.T) {
	var buf bytes.Buffer
	newLogger(&buf, log.Keys{}).Info("order created", log.Event("ORDER_CREATED"))

	m := decode(t, &buf)
	assert.Equal(t, "INFO", m["level"])
	assert.Equal(t, "order-service", m["service"])
	assert.Equal(t, "test", m["environment"])
	assert.Equal(t, "order created", m["message"])
	assert.Equal(t, "ORDER_CREATED", m["event"])
	assert.NotEmpty(t, m["timestamp"])
}

func TestLogrusErrorContract(t *testing.T) {
	var buf bytes.Buffer
	newLogger(&buf, log.Keys{}).Error(log.Wrap(errors.New("gateway timeout")), "failed", log.RequestID("req-1"))

	m := decode(t, &buf)
	assert.Equal(t, "ERROR", m["level"])
	assert.Equal(t, "req-1", m["request_id"])
	assert.Equal(t, "errorString", m["error_type"])
	assert.Equal(t, "gateway timeout", m["error_message"])
	assert.NotEmpty(t, m["stack_trace"])

	stack, ok := m["stack"].([]any)
	require.True(t, ok, "stack should be a JSON array")
	require.NotEmpty(t, stack)
}

func TestLogrusCustomKeys(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(&buf, log.Keys{ErrorMessage: "error", RequestID: "trace_id"})
	l.Error(log.Wrap(errors.New("boom")), "failed", log.RequestID("req-9"))

	m := decode(t, &buf)
	assert.Equal(t, "boom", m["error"])
	assert.Equal(t, "req-9", m["trace_id"])
	assert.NotContains(t, m, "error_message")
	assert.NotContains(t, m, "request_id")
}

func TestLogrusDisabledLevelDropped(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(log.Config{Output: &buf, Level: log.InfoLevel, Driver: Driver})
	l.Debug("dropped")
	assert.Empty(t, buf.String())
}
