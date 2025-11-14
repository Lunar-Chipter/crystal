package rotation

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// compressFile compresses a file using gzip
func (r *RotatingFileWriter) compressFile(srcFilename, dstFilename string) error {
	// Check if source file exists
	if _, err := os.Stat(srcFilename); os.IsNotExist(err) {
		return nil // File already compressed or removed
	}

	srcFile, err := os.Open(srcFilename)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstFilename)
	if err != nil {
		return err
	}
	defer func() {
		dstFile.Close()
		// If compression fails, remove the incomplete compressed file
		if err != nil {
			os.Remove(dstFilename)
		}
	}()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	buf := r.bufferPool.Get().([]byte)
	defer r.bufferPool.Put(buf)

	_, err = io.CopyBuffer(gzWriter, srcFile, buf)
	if err != nil {
		return err
	}

	// Close gzip writer to ensure all data is written
	if err = gzWriter.Close(); err != nil {
		return err
	}

	// Close destination file before removing source
	dstFile.Close()

	// Remove source file after successful compression
	return os.Remove(srcFilename)
}

// cleanupBackups removes old backup files
func (r *RotatingFileWriter) cleanupBackups() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.maxBackups <= 0 {
		return nil
	}

	// Get directory of the log file
	dir := filepath.Dir(r.filename)
	pattern := filepath.Join(dir, "log.*.log*")
	
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
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

	// Count log files (not compressed)
	logFiles := make([]string, 0, len(files))
	for _, file := range files {
		if strings.HasSuffix(file, ".log") && !strings.HasSuffix(file, r.compressedExt) {
			logFiles = append(logFiles, file)
		}
	}

	// Remove old files
	if len(logFiles) > r.maxBackups {
		for i := 0; i < len(logFiles)-r.maxBackups; i++ {
			if err := os.Remove(logFiles[i]); err != nil {
				// Log error but continue with other files
				continue
			}
		}
	}

	return nil
}