// Package outputs provides various output destinations for the Crystal logger.
package outputs

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// RotatingFileWriter provides log rotation with compression.
type RotatingFileWriter struct {
	filename       string
	maxSize        int64
	maxAge         time.Duration
	maxBackups     int
	localTime      bool
	compress       bool
	pattern        string
	compressedExt  string
	
	file       *os.File
	mu         sync.Mutex
	currentSize int64
	lastRotation time.Time
	
	bufferPool sync.Pool
}

// RotationConfig holds configuration for log rotation.
type RotationConfig struct {
	MaxSize       int64
	MaxAge        time.Duration
	MaxBackups    int
	LocalTime     bool
	Compress      bool
	FilenamePattern string
	RotationTime  time.Duration
}

// NewRotatingFileWriter creates a new rotating file writer.
func NewRotatingFileWriter(filename string, config *RotationConfig) (*RotatingFileWriter, error) {
	r := &RotatingFileWriter{
		filename:      filename,
		maxSize:       config.MaxSize,
		maxAge:        config.MaxAge,
		maxBackups:    config.MaxBackups,
		localTime:     config.LocalTime,
		compress:      config.Compress,
		pattern:       "2006-01-02T15-04-05",
		compressedExt: ".gz",
		bufferPool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 0, 32*1024)
				return &buf
			},
		},
	}
	
	if config.FilenamePattern != "" {
		r.pattern = config.FilenamePattern
	}
	
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	r.file = file
	
	// Get initial file size
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	r.currentSize = stat.Size()
	r.lastRotation = time.Now()
	
	return r, nil
}

// Write writes data to the rotating file.
func (r *RotatingFileWriter) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.needsRotation() {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}
	
	n, err = r.file.Write(p)
	r.currentSize += int64(n)
	return n, err
}

// needsRotation determines if rotation is needed.
func (r *RotatingFileWriter) needsRotation() bool {
	now := time.Now()
	if r.maxSize > 0 && r.currentSize >= r.maxSize {
		return true
	}
	
	if r.maxAge > 0 && !r.lastRotation.IsZero() && now.Sub(r.lastRotation) >= r.maxAge {
		return true
	}
	
	return false
}

// rotate performs the actual file rotation.
func (r *RotatingFileWriter) rotate() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return err
		}
	}
	
	// Generate new filename based on pattern
	var newFilename string
	now := time.Now()
	if !r.localTime {
		now = now.UTC()
	}
	
	// Use the filename pattern or default pattern
	pattern := r.pattern
	if pattern == "" {
		pattern = "2006-01-02T15-04-05"
	}
	
	// Format the timestamp
	timestamp := now.Format(pattern)
	
	// Create the new filename
	dir := filepath.Dir(r.filename)
	base := filepath.Base(r.filename)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	
	newFilename = filepath.Join(dir, fmt.Sprintf("%s.%s%s", name, timestamp, ext))
	
	// Compress if needed
	if r.compress {
		newFilename += r.compressedExt
		if err := r.compressFile(r.filename, newFilename); err != nil {
			// If compression fails, try to rename without compression
			if renameErr := os.Rename(r.filename, newFilename[:len(newFilename)-len(r.compressedExt)]); renameErr != nil {
				return fmt.Errorf("failed to compress (%v) and rename (%v)", err, renameErr)
			}
		}
	} else {
		if err := os.Rename(r.filename, newFilename); err != nil {
			return err
		}
	}
	
	// Cleanup old backups
	if err := r.cleanupBackups(); err != nil {
		// Log error but don't fail rotation
		fmt.Fprintf(os.Stderr, "Failed to cleanup backups: %v\n", err)
	}
	
	// Create new file
	file, err := os.OpenFile(r.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	
	r.file = file
	r.currentSize = 0
	r.lastRotation = time.Now()
	return nil
}

// compressFile compresses a file using gzip.
func (r *RotatingFileWriter) compressFile(srcFilename, dstFilename string) error {
	srcFile, err := os.Open(srcFilename)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	
	dstFile, err := os.Create(dstFilename)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	
	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()
	
	bufPtr := r.bufferPool.Get().(*[]byte)
	buf := *bufPtr
	defer r.bufferPool.Put(&buf)
	
	_, err = io.CopyBuffer(gzWriter, srcFile, buf)
	if err != nil {
		return err
	}
	
	return os.Remove(srcFilename)
}

// cleanupBackups removes old backup files.
func (r *RotatingFileWriter) cleanupBackups() error {
	if r.maxBackups <= 0 {
		return nil
	}
	
	// Get directory of the log file
	dir := filepath.Dir(r.filename)
	base := filepath.Base(r.filename)
	
	// Create pattern to match backup files
	pattern := filepath.Join(dir, base+".*.log*")
	
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	
	if len(files) <= r.maxBackups {
		return nil
	}
	
	// Sort files by modification time, oldest first
	sort.Slice(files, func(i, j int) bool {
		info1, err1 := os.Stat(files[i])
		info2, err2 := os.Stat(files[j])
		
		if err1 != nil || err2 != nil {
			return false
		}
		
		return info1.ModTime().Before(info2.ModTime())
	})
	
	// Remove oldest files
	for i := 0; i < len(files)-r.maxBackups; i++ {
		if err := os.Remove(files[i]); err != nil {
			// Log error but continue with other files
			fmt.Fprintf(os.Stderr, "Failed to remove backup file %s: %v\n", files[i], err)
		}
	}
	
	return nil
}

// Close closes the rotating file writer.
func (r *RotatingFileWriter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}