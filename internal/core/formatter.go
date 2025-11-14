package core

// Formatter interface for log formatting with zero-allocation design to minimize garbage collection pressure
// Interface Formatter untuk formatting log dengan desain zero-allocation untuk meminimalkan tekanan garbage collection
type Formatter interface {
	// Format converts a LogEntry into a byte representation with zero allocation where possible
	// Format mengkonversi LogEntry menjadi representasi byte dengan zero allocation jika memungkinkan
	Format(entry interface{}) ([]byte, error)
}