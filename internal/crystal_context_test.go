package crystal

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"crystal/internal/core"
)

func TestContextAwareLogging(t *testing.T) {
	var buf bytes.Buffer
	
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: core.NewTextFormatter(),
	}
	logger := core.NewLogger(config)
	
	// Create context with values
	ctx := context.WithValue(context.Background(), "request_id", "req-abc123")
	ctx = context.WithValue(ctx, "user_id", "user-xyz789")
	
	// Log with context
	logger.InfoContext(ctx, "Processing request")
	
	output := buf.String()
	if !strings.Contains(output, "req-abc123") {
		t.Error("Expected request_id value in context-aware log output")
	}
	
	if !strings.Contains(output, "user-xyz789") {
		t.Error("Expected user_id value in context-aware log output")
	}
}

func TestLogSampling(t *testing.T) {
	var buf bytes.Buffer
	
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: core.NewTextFormatter(),
	}
	logger := core.NewLogger(config)
	
	// TODO: Implement sampling logger tests once sampling functionality is fully implemented
	// This is a placeholder for future sampling tests
	_ = logger
}