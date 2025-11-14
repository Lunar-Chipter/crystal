package interfaces

import (
	"time"
)

// LogEntryInterface defines the interface for log entries
type LogEntryInterface interface {
	GetTimestamp() time.Time
	GetLevel() Level
	GetMessage() string
	GetPID() int
	GetCallerFile() string
	GetCallerLine() int
	GetGoroutineID() string
	GetTraceID() string
	GetSpanID() string
	GetUserID() string
	GetSessionID() string
	GetRequestID() string
	GetDuration() time.Duration
	GetStackTrace() string
	GetHostname() string
	GetApplication() string
	GetVersion() string
	GetEnvironment() string
	GetFields() []interface{}
	GetTags() []string
	GetMetrics() []interface{}
	GetError() error
	// Zero-allocation field access methods
	GetStringField(key string) (string, bool)
	GetIntField(key string) (int, bool)
	GetFloat64Field(key string) (float64, bool)
	GetBoolField(key string) (bool, bool)
}