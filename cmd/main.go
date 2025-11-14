package main

import (
	"context"
	"os"
	"crystal"
	"crystal/internal/core"
)

func main() {
	// Membuat logger default
	logger := crystal.NewDefaultLogger()
	
	// Menguji logging dengan berbagai level
	logger.Info("Ini adalah pesan info")
	logger.Debug("Ini adalah pesan debug")
	logger.Warn("Ini adalah pesan warning")
	logger.Error("Ini adalah pesan error")
	
	// Membuat logger dengan konfigurasi kustom
	config := core.LoggerConfig{
		Level:     core.DEBUG,
		Output:    os.Stdout,
		Formatter: &core.TextFormatter{
			MaskSensitiveData: true,
			MaskString:        "***",
			ShowTimestamp:     true,
			EnableColors:      true,
			TimestampFormat:   "2006-01-02 15:04:05",
		},
	}
	
	customLogger := crystal.NewLogger(config)
	customLogger.Info("Logger kustom berhasil dibuat")
	
	// Test context logging with trace info
	traceConfig := core.LoggerConfig{
		Level:     core.DEBUG,
		Output:    os.Stdout,
		Formatter: &core.TextFormatter{
			MaskSensitiveData: true,
			MaskString:        "***",
			ShowTimestamp:     true,
			EnableColors:      true,
			ShowTraceInfo:     true,
			TimestampFormat:   "2006-01-02 15:04:05",
		},
	}
	
	traceLogger := crystal.NewLogger(traceConfig)
	
	// Test context logging
	ctx := context.WithValue(context.Background(), "request_id", "req-123")
	traceLogger.InfoContext(ctx, "Processing request with context")
	
	// Test sensitive data masking
	customLogger.Info("User login", "username", "john_doe", "password", "secret123")
	
	// Test log injection prevention
	customLogger.Info("User login\\nERROR: Invalid credentials")
}