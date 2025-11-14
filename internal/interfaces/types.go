package interfaces

// FieldPair represents a key-value pair for structured logging fields
type FieldPair struct {
	Key   [64]byte    // Fixed-size key buffer to avoid dynamic allocation for field names
	Value interface{} // Flexible value type to accommodate various data types
	KeyLen int        // Actual key length to track the valid portion of the key buffer
	ValueLen int      // Actual value length for string values to optimize memory usage
}

// MetricPair represents a key-value pair for custom metrics with numeric values
type MetricPair struct {
	Key   [64]byte // Fixed-size key buffer to avoid dynamic allocation for metric names
	Value float64  // Numeric value for performance metrics and measurements
	KeyLen int     // Actual key length to track the valid portion of the key buffer
}