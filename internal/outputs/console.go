// Package outputs provides various output destinations for the Crystal logger.
package outputs

import (
	"io"
	"os"
)

// ConsoleOutput represents a console output destination.
type ConsoleOutput struct {
	writer io.Writer
}

// NewConsoleOutput creates a new console output using stdout.
func NewConsoleOutput() *ConsoleOutput {
	return &ConsoleOutput{
		writer: os.Stdout,
	}
}

// NewConsoleOutputWithWriter creates a new console output with a custom writer.
func NewConsoleOutputWithWriter(writer io.Writer) *ConsoleOutput {
	return &ConsoleOutput{
		writer: writer,
	}
}

// Write writes data to the console output.
func (co *ConsoleOutput) Write(p []byte) (n int, err error) {
	return co.writer.Write(p)
}

// Close closes the console output.
func (co *ConsoleOutput) Close() error {
	if closer, ok := co.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}