package crystal

import (
	"bytes"
	"testing"
	"crystal/internal/core"
)

func TestEndToEndLoggingFlow(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer
	
	// Create logger with buffer output
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: core.NewTextFormatter(),
	}
	logger := core.NewLogger(config)
	
	// Test basic logging flow
	logger.Info("Application started")
	output := buf.String()
	
	if output == "" {
		t.Error("Expected log output, got empty string")
	}
	
	if !bytes.Contains(buf.Bytes(), []byte("Application started")) {
		t.Error("Expected message in log output")
	}
	
	// Reset buffer
	buf.Reset()
	
	// Test structured logging flow
	logger.Error("Database connection failed", "host", "localhost", "port", 5432)
	output = buf.String()
	
	if output == "" {
		t.Error("Expected log output, got empty string")
	}
	
	if !bytes.Contains(buf.Bytes(), []byte("Database connection failed")) {
		t.Error("Expected error message in log output")
	}
	
	if !bytes.Contains(buf.Bytes(), []byte("host")) || !bytes.Contains(buf.Bytes(), []byte("localhost")) {
		t.Error("Expected host field in log output")
	}
}

func TestMultipleLoggersIndependence(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	
	// Create two independent loggers
	config1 := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf1,
		Formatter: core.NewTextFormatter(),
	}
	logger1 := core.NewLogger(config1)
	
	config2 := core.LoggerConfig{
		Level:     core.ERROR,
		Output:    &buf2,
		Formatter: core.NewJSONFormatter(),
	}
	logger2 := core.NewLogger(config2)
	
	// Log to both loggers
	logger1.Info("Info message")
	logger2.Info("This should not appear")
	logger2.Error("Error message")
	
	// Check first logger output
	if !bytes.Contains(buf1.Bytes(), []byte("Info message")) {
		t.Error("Expected info message in first logger output")
	}
	
	// Check second logger output
	if bytes.Contains(buf2.Bytes(), []byte("Info message")) {
		t.Error("Did not expect info message in second logger output")
	}
	
	if !bytes.Contains(buf2.Bytes(), []byte("Error message")) {
		t.Error("Expected error message in second logger output")
	}
}