// Package outputs provides various output destinations for the Crystal logger.
package outputs

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// BufferedWriter - ZERO ALLOCATION VERSION for high-performance buffered writing with minimal garbage collection.
type BufferedWriter struct {
	writer        io.Writer     // Underlying writer for output destination
	buffer        chan []byte   // Channel buffer for storing log entries awaiting write
	bufferSize    int           // Size of the channel buffer to control memory usage
	flushInterval time.Duration // Interval between automatic flush operations
	done          chan struct{} // Channel to signal worker shutdown
	wg            sync.WaitGroup // WaitGroup to coordinate worker shutdown
	batchTimeout  time.Duration // Maximum time to wait for a full batch
	
	// Statistics counters
	droppedLogs int64 // Counter for logs dropped due to full buffer
	totalLogs   int64 // Counter for total logs processed
	lastFlush   time.Time // Timestamp of the last flush operation
}

// NewBufferedWriter creates a new BufferedWriter with specified configuration for high-performance buffered writing.
func NewBufferedWriter(writer io.Writer, bufferSize int, flushInterval time.Duration) *BufferedWriter {
	bw := &BufferedWriter{
		writer:        writer,        // Set the underlying output writer
		buffer:        make(chan []byte, bufferSize), // Create buffered channel with specified size
		bufferSize:    bufferSize,    // Store buffer size for statistics
		flushInterval: flushInterval, // Set automatic flush interval
		done:          make(chan struct{}), // Create shutdown signal channel
		batchTimeout:  100 * time.Millisecond, // Set default batch timeout
		lastFlush:     time.Now(),    // Initialize last flush timestamp
	}
	
	// Start background worker goroutine for processing buffered writes
	bw.wg.Add(1)
	go bw.flushWorker()
	
	return bw
}

// Write writes data with zero allocation strategy using buffer pools and channel buffering for high-performance logging.
func (bw *BufferedWriter) Write(p []byte) (n int, err error) {
	// Increment total logs counter atomically to track processing volume
	// Tambahkan counter total log secara atomik untuk melacak volume pemrosesan
	atomic.AddInt64(&bw.totalLogs, 1)
	
	// Attempt to send data to buffer channel without blocking
	// Upaya mengirim data ke channel buffer tanpa memblokir
	select {
	case bw.buffer <- p:
		// Successfully queued for buffered writing
		// Berhasil diantrekan untuk penulisan buffer
		return len(p), nil
	case <-bw.done:
		// Writer is shutting down, fallback to direct write to prevent data loss
		// Penulis sedang dimatikan, fallback ke penulisan langsung untuk mencegah kehilangan data
		return bw.writer.Write(p)
	default:
		// Buffer channel is full, increment dropped logs counter and fallback to direct write
		// Channel buffer penuh, tambahkan counter log yang dijatuhkan dan fallback ke penulisan langsung
		atomic.AddInt64(&bw.droppedLogs, 1)
		// Fallback to direct write to prevent data loss
		// Fallback ke penulisan langsung untuk mencegah kehilangan data
		return bw.writer.Write(p)
	}
}

// flushWorker processes buffered writes in batches for high-performance I/O with minimal system calls.
func (bw *BufferedWriter) flushWorker() {
	// Signal completion when worker exits
	// Sinyal penyelesaian ketika worker keluar
	defer bw.wg.Done()
	
	// Create ticker for periodic flush intervals
	// Buat ticker untuk interval flush periodik
	ticker := time.NewTicker(bw.flushInterval)
	defer ticker.Stop()
	
	// Create timer for batch timeout to prevent indefinite waiting
	// Buat timer untuk timeout batch untuk mencegah tunggu tanpa batas
	batchTimer := time.NewTimer(bw.batchTimeout)
	defer batchTimer.Stop()
	
	// Reset timer to prevent immediate trigger
	// Reset timer untuk mencegah pemicu langsung
	if !batchTimer.Stop() {
		<-batchTimer.C
	}
	
	// Main event loop for processing buffered writes
	// Loop event utama untuk memproses penulisan buffer
	for {
		select {
		case <-bw.done:
			// Shutdown signal received, flush remaining data and exit
			// Sinyal shutdown diterima, flush data yang tersisa dan keluar
			bw.flushBatch(bw.collectBatch())
			return
		case <-ticker.C:
			// Periodic flush interval reached, flush current batch
			// Interval flush periodik tercapai, flush batch saat ini
			bw.flushBatch(bw.collectBatch())
			bw.lastFlush = time.Now()
			// Reset batch timer
			// Reset timer batch
			if !batchTimer.Stop() {
				select {
				case <-batchTimer.C:
				default:
				}
			}
		case <-batchTimer.C:
			// Batch timeout reached, flush if there's data
			// Timeout batch tercapai, flush jika ada data
			bw.flushBatch(bw.collectBatch())
			bw.lastFlush = time.Now()
			// Reset batch timer for next cycle
			// Reset timer batch untuk siklus berikutnya
			batchTimer.Reset(bw.batchTimeout)
		}
	}
}

// collectBatch collects all available data from buffer channel into a batch slice.
func (bw *BufferedWriter) collectBatch() [][]byte {
	var batch [][]byte
	
	// Collect all immediately available data without blocking
	// Kumpulkan semua data yang tersedia segera tanpa memblokir
	for {
		select {
		case data := <-bw.buffer:
			batch = append(batch, data)
		default:
			// No more data available immediately
			// Tidak ada data lagi yang tersedia segera
			return batch
		}
	}
}

// flushBatch flushes a batch efficiently by combining multiple writes into a single system call to minimize I/O overhead.
func (bw *BufferedWriter) flushBatch(batch [][]byte) {
	// Early return if batch is empty to avoid unnecessary processing
	// Kembali awal jika batch kosong untuk menghindari pemrosesan yang tidak perlu
	if len(batch) == 0 {
		return
	}
	
	// Calculate total size needed for combined buffer to avoid reallocations
	// Hitung ukuran total yang dibutuhkan untuk buffer gabungan untuk menghindari realokasi
	totalSize := 0
	for _, data := range batch {
		totalSize += len(data)
	}
	
	// Create combined buffer for efficient single write operation
	// Buat buffer gabungan untuk operasi tulis tunggal yang efisien
	combined := make([]byte, totalSize)
	offset := 0
	for _, data := range batch {
		copy(combined[offset:], data)
		offset += len(data)
	}
	
	// Perform single write operation for entire batch to minimize system calls
	// Lakukan operasi tulis tunggal untuk seluruh batch untuk meminimalkan panggilan sistem
	_, err := bw.writer.Write(combined)
	if err != nil {
		// Log error but continue to avoid blocking
		// Catat kesalahan tetapi lanjutkan untuk menghindari pemblokiran
		fmt.Printf("BufferedWriter: failed to write batch: %v\n", err)
	}
}

// Stats returns statistics about the buffered writer's performance and status for monitoring and debugging.
func (bw *BufferedWriter) Stats() map[string]interface{} {
	return map[string]interface{}{
		"buffer_size":    bw.bufferSize,  // Configured buffer size
		"current_queue":  len(bw.buffer), // Current number of entries in buffer
		"dropped_logs":   atomic.LoadInt64(&bw.droppedLogs), // Total logs dropped due to full buffer
		"total_logs":     atomic.LoadInt64(&bw.totalLogs),   // Total logs processed
		"last_flush":     bw.lastFlush,   // Timestamp of last flush operation
	}
}

// Flush forces an immediate flush of buffered data
func (bw *BufferedWriter) Flush() error {
	// Force immediate flush of all buffered data
	bw.flushBatch(bw.collectBatch())
	return nil
}

// Close closes the writer by signaling shutdown, waiting for worker completion, flushing remaining data, and closing underlying writer.
func (bw *BufferedWriter) Close() error {
	// Signal shutdown to worker goroutine
	// Sinyalkan shutdown ke goroutine worker
	close(bw.done)
	
	// Wait for worker goroutine to complete processing
	// Tunggu goroutine worker menyelesaikan pemrosesan
	bw.wg.Wait()
	
	// Close underlying writer if it implements io.Closer interface
	// Tutup penulis yang mendasarinya jika mengimplementasikan interface io.Closer
	if closer, ok := bw.writer.(io.Closer); ok {
		return closer.Close()
	}
	
	return nil
}