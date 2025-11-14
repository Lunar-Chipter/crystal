package interfaces

// Level represents the severity level of a log entry
type Level uint8

const (
	// TRACE level for very detailed debugging information, typically used during development
	TRACE Level = iota
	
	// DEBUG level for debugging information, useful for diagnosing problems
	DEBUG
	
	// INFO level for general information about application progress
	INFO
	
	// NOTICE level for normal but significant conditions
	NOTICE
	
	// WARN level for warning conditions that might indicate problems
	WARN
	
	// ERROR level for error conditions that prevent normal operation
	ERROR
	
	// FATAL level for very severe error conditions that will cause the application to exit
	FATAL
	
	// PANIC level for panic conditions that require immediate attention
	PANIC
)

// Pre-computed string representations for zero-allocation access to log levels
var (
	// levelStrings contains pre-allocated string representations of log levels for zero-allocation access
	levelStrings = [...]string{
		"TRACE", "DEBUG", "INFO", "NOTICE", "WARN", "ERROR", "FATAL", "PANIC",
	}
)

// String returns the string representation of the level using zero-allocation technique by accessing pre-computed array
func (l Level) String() string {
	// Zero-allocation access to pre-computed string representations
	if l < Level(len(levelStrings)) {
		return levelStrings[l]
	}
	return "UNKNOWN"
}