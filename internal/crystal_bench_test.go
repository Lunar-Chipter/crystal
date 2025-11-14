package crystal

import (
	"context"
	"testing"

	"crystal/internal/core"
)

type discardWriter struct{}

func (d discardWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func BenchmarkLogger(b *testing.B) {
	config := core.LoggerConfig{
		Level:            core.INFO,
		EnableColors:     false,
		Output:           discardWriter{},
		ErrorOutput:      discardWriter{},
		DisableLocking:   true,
		MaxMessageSize:   core.MAX_MESSAGE_SIZE,
		DisableCallerInfo: false,
		Formatter:        core.NewTextFormatter(),
	}
	
	config.Formatter = &core.TextFormatter{
		EnableColors:     false,
	}
	
	logger := core.NewLogger(config)
	
	b.Run("Simple", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info("test message")
		}
	})
	
	b.Run("WithFields", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info("user action", "user_id", 12345, "action", "login", "ip_address", "192.168.1.1")
		}
	})
	
	b.Run("WithContext", func(b *testing.B) {
		ctx := context.WithValue(context.Background(), "trace_id", "trace-123")
		ctx = context.WithValue(ctx, "span_id", "span-456")
		ctx = context.WithValue(ctx, "user_id", "user-789")
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.InfoContext(ctx, "test message")
		}
	})
	
	b.Run("JSONFormatter", func(b *testing.B) {
		jsonConfig := config
		jsonConfig.Formatter = core.NewJSONFormatter()
		jsonLogger := core.NewLogger(jsonConfig)
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			jsonLogger.Info("test message")
		}
	})
	
	b.Run("ZeroAllocationStringFields", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info("user action", "user_id", "12345", "action", "login", "ip_address", "192.168.1.1")
		}
	})
	
	b.Run("ZeroAllocationIntFields", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info("user action", "user_id", 12345, "status", 200, "count", 1)
		}
	})
	
	b.Run("ZeroAllocationMixedFields", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info("user action", "user_id", "12345", "status", 200, "success", true, "rate", 0.95)
		}
	})
}