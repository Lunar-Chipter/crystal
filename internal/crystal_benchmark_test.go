package crystal

import (
	"bytes"
	"context"
	"testing"
	"crystal/internal/core"
)

// Benchmark tests comparing Crystal with other logging libraries
// These benchmarks test various scenarios to compare performance

func BenchmarkCrystalTextFormatter(b *testing.B) {
	var buf bytes.Buffer
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: core.NewTextFormatter(),
	}
	logger := core.NewLogger(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Test message")
		}
	})
}

func BenchmarkCrystalJSONFormatter(b *testing.B) {
	var buf bytes.Buffer
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: core.NewJSONFormatter(),
	}
	logger := core.NewLogger(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Test message")
		}
	})
}

func BenchmarkCrystalWithFields(b *testing.B) {
	var buf bytes.Buffer
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: core.NewTextFormatter(),
	}
	logger := core.NewLogger(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Test message", "key1", "value1", "key2", 42, "key3", true)
		}
	})
}

func BenchmarkCrystalContextLogging(b *testing.B) {
	var buf bytes.Buffer
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: core.NewTextFormatter(),
	}
	logger := core.NewLogger(config)

	// Create a context with values
	ctx := context.WithValue(context.Background(), "request_id", "req-123")
	ctx = context.WithValue(ctx, "user_id", "user-456")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.InfoContext(ctx, "Test message")
		}
	})
}

func BenchmarkCrystalHighConcurrency(b *testing.B) {
	var buf bytes.Buffer
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: core.NewTextFormatter(),
	}
	logger := core.NewLogger(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("High concurrency test message")
		}
	})
}

func BenchmarkCrystalErrorLogging(b *testing.B) {
	var buf bytes.Buffer
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: core.NewTextFormatter(),
	}
	logger := core.NewLogger(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Error("Error message", "error", "test error")
		}
	})
}

func BenchmarkCrystalTimestampFormatting(b *testing.B) {
	var buf bytes.Buffer
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: core.NewTextFormatter(),
	}
	logger := core.NewLogger(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Timestamp test message")
		}
	})
}

// Benchmark tests for memory allocation
func BenchmarkCrystalMemoryAllocation(b *testing.B) {
	var buf bytes.Buffer
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: core.NewTextFormatter(),
	}
	logger := core.NewLogger(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Memory allocation test")
		}
	})
}