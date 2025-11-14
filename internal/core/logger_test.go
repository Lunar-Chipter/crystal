package core

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// Mock writer for testing
type mockWriter struct {
	buf bytes.Buffer
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *mockWriter) String() string {
	return w.buf.String()
}

func TestLoggerCreation(t *testing.T) {
	t.Run("NewDefaultLogger", func(t *testing.T) {
		logger := NewDefaultLogger()
		if logger == nil {
			t.Error("Expected logger to be created, got nil")
		}
	})

	t.Run("NewLogger", func(t *testing.T) {
		writer := &mockWriter{}
		config := LoggerConfig{
			Level:     INFO,
			Output:    writer,
			Formatter: NewTextFormatter(),
		}
		logger := NewLogger(config)
		if logger == nil {
			t.Error("Expected logger to be created, got nil")
		}
	})
}

func TestLogLevelFiltering(t *testing.T) {
	writer := &mockWriter{}
	config := LoggerConfig{
		Level:     WARN,
		Output:    writer,
		Formatter: NewTextFormatter(),
	}
	logger := NewLogger(config)

	logger.Info("This should not appear")
	if writer.String() != "" {
		t.Error("Expected no output for filtered log level")
	}

	writer.buf.Reset()
	logger.Warn("This should appear")
	if writer.String() == "" {
		t.Error("Expected output for allowed log level")
	}
}

func TestLogMessageFormatting(t *testing.T) {
	writer := &mockWriter{}
	config := LoggerConfig{
		Level:     INFO,
		Output:    writer,
		Formatter: NewTextFormatter(),
	}
	logger := NewLogger(config)

	logger.Info("Test message")
	output := writer.String()
	if !strings.Contains(output, "Test message") {
		t.Error("Expected message to be formatted correctly")
	}
}

func TestLogWithFields(t *testing.T) {
	writer := &mockWriter{}
	config := LoggerConfig{
		Level:     INFO,
		Output:    writer,
		Formatter: NewTextFormatter(),
	}
	logger := NewLogger(config)

	logger.Info("User action", "user_id", 12345, "action", "login")
	output := writer.String()
	if !strings.Contains(output, "user_id") || !strings.Contains(output, "12345") {
		t.Error("Expected fields to be included in log output")
	}
}

func TestContextLogging(t *testing.T) {
	writer := &mockWriter{}
	config := LoggerConfig{
		Level:     INFO,
		Output:    writer,
		Formatter: &TextFormatter{
			ShowTraceInfo: true, // Enable trace info to show context values
		},
	}
	logger := NewLogger(config)

	ctx := context.WithValue(context.Background(), "request_id", "req-123")
	logger.InfoContext(ctx, "Processing request")
	output := writer.String()
	
	// Since context values are stored in fixed buffers rather than as fields,
	// we need to check for the value directly in the output
	if !strings.Contains(output, "req-123") {
		t.Error("Expected context values to be included in log output")
	}
}

func TestErrorLogging(t *testing.T) {
	writer := &mockWriter{}
	config := LoggerConfig{
		Level:     INFO,
		Output:    writer,
		Formatter: NewTextFormatter(),
	}
	logger := NewLogger(config)

	err := errors.New("test error")
	logger.Error("Operation failed", "error", err)
	output := writer.String()
	if !strings.Contains(output, "test error") {
		t.Error("Expected error to be included in log output")
	}
}

func TestTimeField(t *testing.T) {
	writer := &mockWriter{}
	config := LoggerConfig{
		Level:     INFO,
		Output:    writer,
		Formatter: NewTextFormatter(),
	}
	logger := NewLogger(config)

	now := time.Now()
	logger.Info("Timestamp test", "timestamp", now)
	output := writer.String()
	
	// Since time values are formatted differently by the formatter,
	// we'll check for a substring of the timestamp
	if !strings.Contains(output, now.Format("2006-01-02")) {
		t.Error("Expected timestamp to be formatted correctly")
	}
}