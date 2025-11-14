package sampling

import (
	"sync/atomic"
)

// ===============================
// SAMPLING LOGGER - OPTIMIZED
// ===============================

// SamplingLogger provides log sampling to reduce volume
type SamplingLogger struct {
	rate    int
	counter int64
}

// NewSamplingLogger creates a new SamplingLogger
func NewSamplingLogger(rate int) *SamplingLogger {
	return &SamplingLogger{
		rate:   rate,
	}
}

// ShouldLog determines if a log should be recorded based on sampling rate
func (sl *SamplingLogger) ShouldLog() bool {
	// If rate is 0 or negative, don't log anything
	if sl.rate <= 0 {
		return false
	}
	
	// If rate is 1, log everything
	if sl.rate == 1 {
		return true
	}
	
	// Increment counter atomically and check if we should log
	counter := atomic.AddInt64(&sl.counter, 1)
	return counter%int64(sl.rate) == 0
}