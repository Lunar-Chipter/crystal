// Package crystal provides a high-performance, zero-allocation logging framework designed for production systems
// Paket crystal menyediakan kerangka kerja logging berkinerja tinggi dengan desain zero-allocation untuk sistem produksi
package crystal

import (
	"crystal/internal/core"
	"crystal/internal/outputs"
	"crystal/internal/rotation"
	"crystal/internal/sampling"
	"crystal/internal/metrics"
	"crystal/internal/config"
	"crystal/internal/interfaces"
)

// Type aliases for backward compatibility
type Logger = core.Logger
type LogEntry = core.LogEntry
type Level = core.Level
type Formatter = core.Formatter
type TextFormatter = core.TextFormatter
type JSONFormatter = core.JSONFormatter
type BufferedWriter = outputs.BufferedWriter
type RotatingFileWriter = rotation.RotatingFileWriter
type SamplingLogger = sampling.SamplingLogger
type LoggerConfig = config.LoggerConfig
type RotationConfig = config.RotationConfig
type LogEntryInterface = interfaces.LogEntryInterface
type FieldPair = interfaces.FieldPair
type MetricPair = interfaces.MetricPair

// Level constants
const (
	TRACE  = core.TRACE
	DEBUG  = core.DEBUG
	INFO   = core.INFO
	NOTICE = core.NOTICE
	WARN   = core.WARN
	ERROR  = core.ERROR
	FATAL  = core.FATAL
	PANIC  = core.PANIC
)

// Convenience functions
var (
	NewDefaultLogger      = core.NewDefaultLogger
	NewLogger             = core.NewLogger
	NewTextFormatter      = core.NewTextFormatter
	NewJSONFormatter      = core.NewJSONFormatter
	NewBufferedWriter     = outputs.NewBufferedWriter
	NewRotatingFileWriter = rotation.NewRotatingFileWriter
	NewSamplingLogger     = sampling.NewSamplingLogger
	NewDefaultMetricsCollector = metrics.NewDefaultMetricsCollector
)

// ParseLevel parses a string representation into a Level constant
func ParseLevel(levelStr string) (Level, error) {
	return core.ParseLevel(levelStr)
}