package config

import (
	"time"
)

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	// Output configuration
	OutputFile    string
	OutputFormat  string // "text", "json", "csv"
	Buffered      bool
	BufferSize    int
	FlushInterval time.Duration
	
	// Rotation configuration
	RotationConfig *RotationConfig
	
	// Sampling configuration
	SamplingRate int
	
	// Feature flags
	DisableTimestamp   bool
	DisableColors      bool
	DisableLocking     bool
	EnableSampling     bool
	EnableMetrics      bool
	EnableStackTrace   bool
	EnableContext      bool
	
	// Performance settings
	MaxGoroutines int
	
	// Custom handlers
	ErrorHandler func(error)
	OnFatal      func(map[string]interface{})
	OnPanic      func(map[string]interface{})
	
	// Context extraction
	ContextExtractor func(interface{}) map[string]string
}

// RotationConfig holds configuration for log rotation
type RotationConfig struct {
	MaxSize       int64
	MaxAge        time.Duration
	MaxBackups    int
	LocalTime     bool
	Compress      bool
	RotationTime  time.Duration
	FilenamePattern string
}

// NewDefaultConfig creates a new default logger configuration
func NewDefaultConfig() *LoggerConfig {
	return &LoggerConfig{
		OutputFormat:   "text",
		Buffered:       true,
		BufferSize:     1000,
		FlushInterval:  5 * time.Second,
		RotationConfig: &RotationConfig{},
		SamplingRate:   1,
		MaxGoroutines:  1000,
	}
}