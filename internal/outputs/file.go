// Package outputs provides various output destinations for the Crystal logger.
package outputs

import (
	"os"
)

// FileOutput represents a file output destination.
type FileOutput struct {
	file *os.File
}

// NewFileOutput creates a new file output destination.
func NewFileOutput(filename string) (*FileOutput, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	
	return &FileOutput{
		file: file,
	}, nil
}

// Write writes data to the file output.
func (fo *FileOutput) Write(p []byte) (n int, err error) {
	return fo.file.Write(p)
}

// Close closes the file output.
func (fo *FileOutput) Close() error {
	if fo.file != nil {
		return fo.file.Close()
	}
	return nil
}