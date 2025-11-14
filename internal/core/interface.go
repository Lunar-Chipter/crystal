package core

import "time"

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
}

// Ensure LogEntry implements LogEntryInterface
var _ LogEntryInterface = (*LogEntry)(nil)