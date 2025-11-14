package rotation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RotationConfig holds configuration for log rotation
type RotationConfig struct {
	MaxSize         int64
	MaxAge          time.Duration
	MaxBackups      int
	LocalTime       bool
	Compress        bool
	RotationTime    time.Duration
	FilenamePattern string
}

// RotatingFileWriter provides log rotation with compression
type RotatingFileWriter struct {
	filename       string
	maxSize        int64
	maxAge         time.Duration
	maxBackups     int
	localTime      bool
	compress       bool
	rotationTime   time.Duration
	currentSize    int64
	file           *os.File
	mu             sync.Mutex
	lastRotation   time.Time
	pattern        string
	compressedExt  string
	bufferPool     sync.Pool
}

// NewRotatingFileWriter creates a new rotating file writer
func NewRotatingFileWriter(filename string, config *RotationConfig) (*RotatingFileWriter, error) {
	r := &RotatingFileWriter{
		filename:      filename,
		compress:      config.Compress,
		compressedExt: ".gz",
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 64*1024)
			},
		},
	}

	if config.MaxSize > 0 {
		r.maxSize = config.MaxSize
	}

	if config.MaxAge > 0 {
		r.maxAge = config.MaxAge
	}

	if config.MaxBackups > 0 {
		r.maxBackups = config.MaxBackups
	}

	if config.LocalTime {
		r.localTime = config.LocalTime
	}

	if config.RotationTime > 0 {
		r.rotationTime = config.RotationTime
	}

	if config.FilenamePattern != "" {
		r.pattern = filepath.Join(filepath.Dir(filename), config.FilenamePattern)
	} else {
		r.pattern = filepath.Join(filepath.Dir(filename), "log.2006-01-02T15-04-05.000.log")
	}

	info, err := os.Stat(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		r.file, err = os.Create(filename)
		if err != nil {
			return nil, err
		}
		r.lastRotation = time.Now()
		return r, nil
	}

	r.file, err = os.OpenFile(filename, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	r.currentSize = info.Size()
	r.lastRotation = time.Now()
	return r, nil
}

// Write writes data to the rotating file
func (r *RotatingFileWriter) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.needsRotation() {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = r.file.Write(p)
	if err != nil {
		return n, err
	}

	r.currentSize += int64(n)
	return n, nil
}

// needsRotation determines if rotation is needed
func (r *RotatingFileWriter) needsRotation() bool {
	now := time.Now()

	if r.maxSize > 0 && r.currentSize >= r.maxSize {
		return true
	}

	if r.rotationTime > 0 && now.Sub(r.lastRotation) >= r.rotationTime {
		return true
	}

	if r.maxAge > 0 && !r.lastRotation.IsZero() && now.Sub(r.lastRotation) >= r.maxAge {
		return true
	}

	return false
}

// rotate performs the actual file rotation
func (r *RotatingFileWriter) rotate() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return err
		}
	}

	var newFilename string
	if r.localTime {
		newFilename = time.Now().Format(r.pattern)
	} else {
		newFilename = time.Now().UTC().Format(r.pattern)
	}

	// Check if target file already exists
	if _, err := os.Stat(newFilename); err == nil {
		// File exists, add timestamp to make it unique
		ext := filepath.Ext(newFilename)
		base := strings.TrimSuffix(newFilename, ext)
		newFilename = fmt.Sprintf("%s_%d%s", base, time.Now().UnixNano(), ext)
	}

	if err := os.Rename(r.filename, newFilename); err != nil {
		return err
	}

	var err error
	r.file, err = os.Create(r.filename)
	if err != nil {
		return err
	}

	r.currentSize = 0
	r.lastRotation = time.Now()

	if r.compress {
		go func() {
			compressedName := newFilename + r.compressedExt
			if err := r.compressFile(newFilename, compressedName); err != nil {
				// Log error but don't block rotation
				return
			}
		}()
	}

	if r.maxBackups > 0 {
		go r.cleanupBackups()
	}

	return nil
}

// Close closes the rotating file writer
func (r *RotatingFileWriter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		return r.file.Close()
	}
	return nil
}