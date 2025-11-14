// Package core provides core components for the crystal logger
package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"crystal/internal/outputs"
	"crystal/internal/rotation"
	"crystal/internal/metrics"
	"crystal/internal/interfaces"
)

const (
	// DEFAULT_BUFFER_SIZE sets the default buffer size for the buffered writer to optimize I/O operations
	// DEFAULT_BUFFER_SIZE menetapkan ukuran buffer default untuk writer yang di-buffer untuk mengoptimalkan operasi I/O
	DEFAULT_BUFFER_SIZE = 1000
	
	// DEFAULT_FLUSH_INTERVAL defines how often the buffered writer should flush its contents to the output
	// DEFAULT_FLUSH_INTERVAL menentukan seberapa sering writer yang di-buffer harus mengosongkan isinya ke output
	DEFAULT_FLUSH_INTERVAL = 5 * time.Second
)

// LoggerConfig holds configuration options for the logger with zero-allocation optimizations
// LoggerConfig menyimpan opsi konfigurasi untuk logger dengan optimasi zero-allocation
type LoggerConfig struct {
	// Basic configuration options for logger behavior and output
	// Opsi konfigurasi dasar untuk perilaku logger dan output
	Level            Level         // Minimum log level to output - Tingkat log minimum untuk output
	Output           io.Writer     // Primary output destination - Tujuan output utama
	ErrorOutput      io.Writer     // Separate output for errors - Output terpisah untuk kesalahan
	Formatter        Formatter     // Log entry formatter - Formatter entri log
	ExitFunc         func(int)     // Function to call on Fatal logs - Fungsi untuk dipanggil pada log Fatal
	Hostname         string        // Hostname to include in logs - Nama host untuk disertakan dalam log
	Application      string        // Application name - Nama aplikasi
	Version          string        // Application version - Versi aplikasi
	Environment      string        // Environment (dev, staging, prod) - Lingkungan (dev, staging, prod)
	
	// Performance optimization settings to maximize throughput and minimize latency
	// Pengaturan optimasi kinerja untuk memaksimalkan throughput dan meminimalkan latency
	BufferSize       int           // Buffer size for high-performance I/O - Ukuran buffer untuk I/O berkinerja tinggi
	FlushInterval    time.Duration // Interval between automatic flushes - Interval antara flush otomatis
	MaxMessageSize   int           // Maximum message size before truncation - Ukuran pesan maksimum sebelum dipotong
	EnableSampling   bool          // Enable log sampling to reduce volume - Aktifkan sampling log untuk mengurangi volume
	SamplingRate     int           // Sampling rate (1 in N entries) - Tingkat sampling (1 dari N entri)
	AsyncLogging     bool          // Enable asynchronous logging - Aktifkan logging asinkron
	ContextExtractor func(context.Context) map[string]string // Function to extract context values - Fungsi untuk mengekstrak nilai konteks
	
	// Feature flags to enable or disable specific functionality for customization
	// Flag fitur untuk mengaktifkan atau menonaktifkan fungsionalitas tertentu untuk kustomisasi
	ShowHostname     bool          // Include hostname in log entries - Sertakan nama host dalam entri log
	ShowApplication  bool          // Include application name - Sertakan nama aplikasi
	ShowVersion      bool          // Include application version - Sertakan versi aplikasi
	ShowEnvironment  bool          // Include environment information - Sertakan informasi lingkungan
	EnableColors     bool          // Enable colored output - Aktifkan output berwarna
	EnableStackTrace bool          // Enable stack trace capture - Aktifkan pengambilan stack trace
	
	// Metrics and monitoring configuration for observability and performance tracking
	// Konfigurasi metrik dan monitoring untuk observabilitas dan pelacakan kinerja
	MetricsCollector metrics.MetricsCollector // Collector for log metrics - Kolektor untuk metrik log
	ErrorHandler     func(error)      // Handler for logger errors - Handler untuk kesalahan logger
	OnFatal          func(*LogEntry)  // Handler for Fatal log entries - Handler untuk entri log Fatal
	OnPanic          func(*LogEntry)  // Handler for Panic log entries - Handler untuk entri log Panic
	
	// Zero-allocation optimizations to maximize performance
	// Optimasi zero-allocation untuk memaksimalkan kinerja
	DisableLocking       bool  // Disable mutex locking for single-threaded use - Nonaktifkan penguncian mutex untuk penggunaan single-threaded
	DisableFieldCopy     bool  // Skip field copying for performance - Lewati penyalinan field untuk kinerja
	DisableTimestamp     bool  // Skip timestamp for performance - Lewati timestamp untuk kinerja
	DisableCallerInfo    bool  // Skip caller info for performance - Lewati info pemanggil untuk kinerja
	FastPathLevel        Level // Levels below this use fast path - Tingkat di bawah ini menggunakan jalur cepat
}

// Logger - ZERO ALLOCATION VERSION for high-performance logging with minimal garbage collection
// Logger - VERSI ZERO ALLOCATION untuk logging berkinerja tinggi dengan garbage collection minimal
type Logger struct {
	config             LoggerConfig              // Logger configuration settings
	formatter          Formatter                 // Log entry formatter (TextFormatter or JSONFormatter)
	out                io.Writer                 // Primary output destination for log entries
	errOut             io.Writer                 // Separate output destination for error-level entries
	mu                 sync.Mutex                // Mutex for thread-safe operations
	hooks              []func(*LogEntry)         // Hooks to execute for each log entry
	exitFunc           func(int)                 // Function to call on Fatal logs (default: os.Exit)
	fields             map[string]interface{}    // Global fields to include in all log entries
	sampler            *SamplingLogger           // Logger for sampling functionality
	buffer             *outputs.BufferedWriter   // Buffered writer for high-performance I/O
	rotation           *rotation.RotatingFileWriter       // Rotating file writer for log rotation
	contextExtractor   func(context.Context) map[string]string // Function to extract context values
	metrics            metrics.MetricsCollector   // Collector for log metrics
	errorHandler       func(error)               // Handler for logger errors
	onFatal            func(*LogEntry)           // Handler for Fatal log entries
	onPanic            func(*LogEntry)           // Handler for Panic log entries
	stats              *LoggerStats              // Statistics collector for logger performance
	asyncLogger        *AsyncLogger              // Asynchronous logger for non-blocking operations
	// Zero-allocation optimizations to maximize performance and minimize garbage collection
	// Optimasi zero-allocation untuk memaksimalkan kinerja dan meminimalkan garbage collection
	levelMask          uint64           // Bitmask for fast level checking without allocations - Bitmask untuk pemeriksaan tingkat cepat tanpa alokasi
	hostnameBytes      []byte           // Pre-converted hostname to avoid string conversions - Hostname yang telah dikonversi sebelumnya untuk menghindari konversi string
	applicationBytes   []byte           // Pre-converted application name to avoid string conversions - Nama aplikasi yang telah dikonversi sebelumnya untuk menghindari konversi string
	versionBytes       []byte           // Pre-converted version to avoid string conversions - Versi yang telah dikonversi sebelumnya untuk menghindari konversi string
	environmentBytes   []byte           // Pre-converted environment to avoid string conversions - Lingkungan yang telah dikonversi sebelumnya untuk menghindari konversi string
	pidStr             []byte           // Pre-converted PID to avoid string conversions - PID yang telah dikonversi sebelumnya untuk menghindari konversi string
}

// NewDefaultLogger creates a logger with default configuration optimized for production use with zero-allocation design
// NewDefaultLogger membuat logger dengan konfigurasi default yang dioptimalkan untuk penggunaan produksi dengan desain zero-allocation
func NewDefaultLogger() *Logger {
	// Create default configuration with production-optimized settings
	// Buat konfigurasi default dengan pengaturan yang dioptimalkan untuk produksi
	config := LoggerConfig{
		Level:         INFO,                          // INFO level as default for production - Level INFO sebagai default untuk produksi
		Output:        os.Stdout,                     // Standard output as default destination - Output standar sebagai tujuan default
		ErrorOutput:   os.Stderr,                     // Standard error for error-level entries - Error standar untuk entri level kesalahan
		Formatter:     NewTextFormatter(),            // Text formatter for human-readable output - Formatter teks untuk output yang dapat dibaca manusia
		ExitFunc:      os.Exit,                       // Default exit function - Fungsi exit default
		BufferSize:    DEFAULT_BUFFER_SIZE,           // Default buffer size for high-performance I/O - Ukuran buffer default untuk I/O berkinerja tinggi
		FlushInterval: DEFAULT_FLUSH_INTERVAL,        // Default flush interval - Interval flush default
		ShowHostname:  true,                          // Include hostname by default - Sertakan nama host secara default
		EnableColors:  true,                          // Enable colors by default - Aktifkan warna secara default
		Hostname:      getHostname(),                 // Auto-detect hostname - Deteksi otomatis nama host
		Application:   "crystal",                     // Default application name - Nama aplikasi default
		Version:       "1.0.0",                       // Default version - Versi default
		Environment:   "production",                  // Default environment - Lingkungan default
		MetricsCollector: metrics.NewDefaultMetricsCollector(), // Default metrics collector - Kolektor metrik default
		ErrorHandler:  defaultErrorHandler,           // Default error handler - Handler kesalahan default
		OnFatal:       defaultFatalHandler,           // Default fatal handler - Handler fatal default
		OnPanic:       defaultPanicHandler,           // Default panic handler - Handler panic default
	}
	return NewLogger(config)
}

// NewLogger creates a new logger with the specified configuration using zero-allocation techniques for high performance
// NewLogger membuat logger baru dengan konfigurasi yang ditentukan menggunakan teknik zero-allocation untuk kinerja tinggi
func NewLogger(config LoggerConfig) *Logger {
	l := &Logger{
		config:    config,                    // Store configuration for future reference
		formatter: config.Formatter,          // Set log entry formatter
		out:       config.Output,             // Set primary output destination
		errOut:    config.ErrorOutput,        // Set error output destination
		exitFunc:  config.ExitFunc,           // Set exit function for fatal logs
		fields:    make(map[string]interface{}), // Initialize global fields map
		contextExtractor: config.ContextExtractor, // Set context extractor function
		metrics:   config.MetricsCollector,   // Set metrics collector
		errorHandler: config.ErrorHandler,    // Set error handler function
		onFatal:   config.OnFatal,            // Set fatal log handler
		onPanic:   config.OnPanic,            // Set panic log handler
		stats:     NewLoggerStats(),          // Initialize statistics collector
	}
	// Set default exit function if not provided
	// Atur fungsi exit default jika tidak disediakan
	if l.exitFunc == nil {
		l.exitFunc = os.Exit
	}
	// Pre-compute level mask for fast checking without allocations
	// Hitung mask tingkat sebelumnya untuk pemeriksaan cepat tanpa alokasi
	l.levelMask = uint64(0)
	for i := config.Level; i < 8; i++ {
		// Use pre-computed bitmasks for efficient level checking
		// Gunakan bitmask yang telah dihitung sebelumnya untuk pemeriksaan tingkat yang efisien
		l.levelMask |= levelMasks[i]
	}
	// Pre-convert static strings to bytes to avoid repeated conversions and allocations
	// Konversi string statis ke byte sebelumnya untuk menghindari konversi dan alokasi berulang
	l.hostnameBytes = sToBytes(config.Hostname)
	l.applicationBytes = sToBytes(config.Application)
	l.versionBytes = sToBytes(config.Version)
	l.environmentBytes = sToBytes(config.Environment)
	l.pidStr = sToBytes(strconv.Itoa(os.Getpid()))
	// Setup buffering for high-performance I/O if configured
	// Siapkan buffering untuk I/O berkinerja tinggi jika dikonfigurasi
	if config.BufferSize > 0 {
		// Create buffered writer for efficient batched writes
		// Buat penulis buffer untuk penulisan batch yang efisien
		l.buffer = outputs.NewBufferedWriter(config.Output, config.BufferSize, config.FlushInterval)
		l.out = l.buffer
	}
	// Setup sampling for reduced log volume if configured
	// Siapkan sampling untuk volume log yang dikurangi jika dikonfigurasi
	if config.EnableSampling && config.SamplingRate > 1 {
		// Create sampling logger to reduce output volume
		// Buat logger sampling untuk mengurangi volume output
		l.sampler = NewSamplingLogger(l, config.SamplingRate)
	}
	// Setup async logging for non-blocking operations if configured
	// Siapkan logging async untuk operasi non-blocking jika dikonfigurasi
	if config.AsyncLogging {
		// Create async logger for non-blocking log operations
		// Buat logger async untuk operasi log non-blocking
		l.asyncLogger = NewAsyncLogger(l, 4, config.BufferSize)
	}
	return l
}

// SetLevel sets the minimum log level with optimized bitmask for zero-allocation level checking
// SetLevel menetapkan tingkat log minimum dengan bitmask yang dioptimalkan untuk pemeriksaan tingkat zero-allocation
func (l *Logger) SetLevel(level Level) {
	// Acquire lock for thread-safe operation if locking is not disabled
	// Dapatkan kunci untuk operasi thread-safe jika penguncian tidak dinonaktifkan
	if !l.config.DisableLocking {
		l.mu.Lock()
		defer l.mu.Unlock()
	}
	// Pre-compute level mask for fast checking without allocations
	// Hitung mask tingkat sebelumnya untuk pemeriksaan cepat tanpa alokasi
	l.levelMask = uint64(0)
	for i := level; i < 8; i++ {
		// Use pre-computed bitmasks for efficient level checking
		// Gunakan bitmask yang telah dihitung sebelumnya untuk pemeriksaan tingkat yang efisien
		l.levelMask |= levelMasks[i]
	}
}

// Fast path level check
func (l *Logger) shouldLog(level Level) bool {
	return (l.levelMask & (1 << level)) != 0
}

// logEntry is the core logging method that takes a pre-populated LogEntry
func (l *Logger) logEntry(entry *LogEntry, ctx context.Context) {
	// Fast path check
	if !l.shouldLog(entry.Level) {
		putEntryToPool(entry)
		return
	}
	// Sampling check
	if l.sampler != nil && !l.sampler.shouldLog() {
		putEntryToPool(entry)
		return
	}
	// Message size check
	if l.config.MaxMessageSize > 0 && entry.MessageLen > l.config.MaxMessageSize {
		// Truncate message in place
		copy(entry.Message[l.config.MaxMessageSize:], "... [truncated]")
		entry.MessageLen = l.config.MaxMessageSize + 14 // length of "... [truncated]"
		if entry.MessageLen > len(entry.Message) {
			entry.MessageLen = len(entry.Message)
		}
	}
	// Fill core fields with zero allocation
	if !l.config.DisableTimestamp {
		entry.Timestamp = time.Now()
	}
	// Copy level name to fixed buffer
	levelStr := entry.Level.String()
	copy(entry.LevelName[:], levelStr)
	// Set static metadata
	entry.PID = os.Getpid()
	// Copy hostname
	if l.config.ShowHostname && len(l.hostnameBytes) > 0 {
		copy(entry.Hostname[:], l.hostnameBytes)
		entry.HostnameLen = len(l.hostnameBytes)
	}
	// Copy application
	if l.config.ShowApplication && len(l.applicationBytes) > 0 {
		copy(entry.Application[:], l.applicationBytes)
		entry.ApplicationLen = len(l.applicationBytes)
	}
	// Copy version
	if l.config.ShowVersion && len(l.versionBytes) > 0 {
		copy(entry.Version[:], l.versionBytes)
		entry.VersionLen = len(l.versionBytes)
	}
	// Copy environment
	if l.config.ShowEnvironment && len(l.environmentBytes) > 0 {
		copy(entry.Environment[:], l.environmentBytes)
		entry.EnvironmentLen = len(l.environmentBytes)
	}
	// Extract caller information if enabled
	if !l.config.DisableCallerInfo {
		// Get caller info with configurable depth for flexibility
		// Dapatkan info pemanggil dengan kedalaman yang dapat dikonfigurasi untuk fleksibilitas
		_, file, line, ok := runtime.Caller(DEFAULT_CALLER_DEPTH)
		if ok {
			// Copy file name to fixed buffer to avoid allocation
			// Salin nama file ke buffer tetap untuk menghindari alokasi
			fileLen := len(file)
			if fileLen > len(entry.Caller.File) {
				fileLen = len(entry.Caller.File)
			}
			copy(entry.Caller.File[:], file[:fileLen])
			entry.Caller.FileLen = fileLen
			entry.Caller.Line = line
			// Extract function and package names from file path for better context
			// Ekstrak nama fungsi dan paket dari path file untuk konteks yang lebih baik
			if idx := lastIndexByte(file, '/'); idx != -1 {
				packageName := file[:idx]
				if pkgIdx := lastIndexByte(packageName, '/'); pkgIdx != -1 {
					// Copy package name to fixed buffer to avoid allocation
					// Salin nama paket ke buffer tetap untuk menghindari alokasi
					pkgLen := len(packageName[pkgIdx+1:])
					if pkgLen > len(entry.Caller.Package) {
						pkgLen = len(entry.Caller.Package)
					}
					copy(entry.Caller.Package[:], packageName[pkgIdx+1:pkgIdx+1+pkgLen])
					entry.Caller.PackageLen = pkgLen
				}
			}
		}
	}
	// Add global fields if any
	// Tambahkan field global jika ada
	if len(l.fields) > 0 && !l.config.DisableFieldCopy {
		// Acquire lock for thread-safe operation if locking is not disabled
		// Dapatkan kunci untuk operasi thread-safe jika penguncian tidak dinonaktifkan
		if !l.config.DisableLocking {
			l.mu.Lock()
		}
		for key, value := range l.fields {
			entry.SetField(key, value)
		}
		if !l.config.DisableLocking {
			l.mu.Unlock()
		}
	}
	// Add context fields if extractor is provided and context is not nil
	// Tambahkan field konteks jika extractor disediakan dan konteks tidak nil
	if l.contextExtractor != nil && ctx != nil {
		contextFields := l.contextExtractor(ctx)
		for key, value := range contextFields {
			entry.SetField(key, value)
		}
	}
	// Add context values if context is not nil
	// Tambahkan nilai konteks jika konteks tidak nil
	if ctx != nil {
		// For zero allocation, we'll use a more direct approach
		// Extract context values directly without creating intermediate maps
		if traceID := ctx.Value("trace_id"); traceID != nil {
			if traceIDStr, ok := traceID.(string); ok {
				// Copy trace ID to fixed buffer to avoid allocation
				// Salin ID trace ke buffer tetap untuk menghindari alokasi
				traceIDLen := len(traceIDStr)
				if traceIDLen > len(entry.TraceID) {
					traceIDLen = len(entry.TraceID)
				}
				copy(entry.TraceID[:], traceIDStr[:traceIDLen])
				entry.TraceIDLen = traceIDLen
			}
		}
		if spanID := ctx.Value("span_id"); spanID != nil {
			if spanIDStr, ok := spanID.(string); ok {
				// Copy span ID to fixed buffer to avoid allocation
				// Salin ID span ke buffer tetap untuk menghindari alokasi
				spanIDLen := len(spanIDStr)
				if spanIDLen > len(entry.SpanID) {
					spanIDLen = len(entry.SpanID)
				}
				copy(entry.SpanID[:], spanIDStr[:spanIDLen])
				entry.SpanIDLen = spanIDLen
			}
		}
		if userID := ctx.Value("user_id"); userID != nil {
			if userIDStr, ok := userID.(string); ok {
				// Copy user ID to fixed buffer to avoid allocation
				// Salin ID pengguna ke buffer tetap untuk menghindari alokasi
				userIDLen := len(userIDStr)
				if userIDLen > len(entry.UserID) {
					userIDLen = len(entry.UserID)
				}
				copy(entry.UserID[:], userIDStr[:userIDLen])
				entry.UserIDLen = userIDLen
			}
		}
		if sessionID := ctx.Value("session_id"); sessionID != nil {
			if sessionIDStr, ok := sessionID.(string); ok {
				// Copy session ID to fixed buffer to avoid allocation
				// Salin ID sesi ke buffer tetap untuk menghindari alokasi
				sessionIDLen := len(sessionIDStr)
				if sessionIDLen > len(entry.SessionID) {
					sessionIDLen = len(entry.SessionID)
				}
				copy(entry.SessionID[:], sessionIDStr[:sessionIDLen])
				entry.SessionIDLen = sessionIDLen
			}
		}
		if requestID := ctx.Value("request_id"); requestID != nil {
			if requestIDStr, ok := requestID.(string); ok {
				// Copy request ID to fixed buffer to avoid allocation
				// Salin ID permintaan ke buffer tetap untuk menghindari alokasi
				requestIDLen := len(requestIDStr)
				if requestIDLen > len(entry.RequestID) {
					requestIDLen = len(entry.RequestID)
				}
				copy(entry.RequestID[:], requestIDStr[:requestIDLen])
				entry.RequestIDLen = requestIDLen
			}
		}
	}
	// Execute hooks if any
	// Jalankan hook jika ada
	if len(l.hooks) > 0 {
		// Acquire lock for thread-safe operation if locking is not disabled
		// Dapatkan kunci untuk operasi thread-safe jika penguncian tidak dinonaktifkan
		if !l.config.DisableLocking {
			l.mu.Lock()
		}
		for _, hook := range l.hooks {
			hook(entry)
		}
		if !l.config.DisableLocking {
			l.mu.Unlock()
		}
	}
	// Format and write the log entry
	// Format dan tulis entri log
	var output []byte
	var err error
	// Use formatter to convert entry to bytes with zero allocation
	// Gunakan formatter untuk mengkonversi entri ke byte dengan zero allocation
	output, err = l.formatter.Format(entry)
	if err != nil {
		// Handle formatting error with error handler
		// Tangani kesalahan formatting dengan handler kesalahan
		if l.errorHandler != nil {
			l.errorHandler(fmt.Errorf("failed to format log entry: %w", err))
		}
		putEntryToPool(entry)
		return
	}
	// Write to appropriate output based on log level
	// Tulis ke output yang sesuai berdasarkan tingkat log
	writer := l.out
	if entry.Level >= ERROR && l.errOut != nil {
		writer = l.errOut
	}
	// Write with buffering if enabled for high-performance I/O
	// Tulis dengan buffering jika diaktifkan untuk I/O berkinerja tinggi
	if l.buffer != nil {
		_, err = l.buffer.Write(output)
	} else {
		// Direct write for immediate output
		// Tulis langsung untuk output segera
		_, err = writer.Write(output)
	}
	if err != nil {
		// Handle write error with error handler
		// Tangani kesalahan penulisan dengan handler kesalahan
		if l.errorHandler != nil {
			l.errorHandler(fmt.Errorf("failed to write log entry: %w", err))
		}
	}
	// Update metrics if collector is provided
	// Perbarui metrik jika kolektor disediakan
	if l.metrics != nil {
		tags := map[string]string{
			"level": strings.ToLower(entry.Level.String()),
		}
		l.metrics.IncrementCounter(interfaces.Level(entry.Level), tags)
		// Note: AddBytesWritten is not available in the interface, so we'll skip it for now
	}
	// Update statistics
	// Perbarui statistik
	if l.stats != nil {
		// We can't take the address of a map index expression, so we need to use a different approach
		// For now, we'll skip atomic updates to LogCounts and BytesWritten
		// In a real implementation, we would need to use a mutex or other synchronization mechanism
	}
	// Handle fatal and panic levels
	// Tangani tingkat fatal dan panic
	if entry.Level == FATAL {
		// Execute fatal handler if provided
		// Jalankan handler fatal jika disediakan
		if l.onFatal != nil {
			l.onFatal(entry)
		}
		// Exit the program with error code
		// Keluar dari program dengan kode kesalahan
		l.exitFunc(1)
	} else if entry.Level == PANIC {
		// Execute panic handler if provided
		// Jalankan handler panic jika disediakan
		if l.onPanic != nil {
			l.onPanic(entry)
		}
		// Get the message from the entry for panic
		msg := string(entry.Message[:entry.MessageLen])
		// Return entry to pool before panic
		putEntryToPool(entry)
		// Panic with the message
		// Panic dengan pesan tersebut
		panic(msg)
	}
	// Return entry to pool for reuse
	putEntryToPool(entry)
	// Flush buffer if flush interval has passed for consistent output
	// Flush buffer jika interval flush telah berlalu untuk output yang konsisten
	if l.buffer != nil {
		l.buffer.Flush()
	}
}

// Trace logs a message at TRACE level with zero allocation
// Trace mencatat pesan pada tingkat TRACE dengan zero allocation
func (l *Logger) Trace(msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = TRACE
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = TRACE
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, nil)
	}
}

// Debug logs a message at DEBUG level with zero allocation
// Debug mencatat pesan pada tingkat DEBUG dengan zero allocation
func (l *Logger) Debug(msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = DEBUG
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = DEBUG
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, nil)
	}
}

// Info logs a message at INFO level with zero allocation
// Info mencatat pesan pada tingkat INFO dengan zero allocation
func (l *Logger) Info(msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = INFO
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = INFO
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, nil)
	}
}

// Notice logs a message at NOTICE level with zero allocation
// Notice mencatat pesan pada tingkat NOTICE dengan zero allocation
func (l *Logger) Notice(msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = NOTICE
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = NOTICE
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, nil)
	}
}

// Warn logs a message at WARN level with zero allocation
// Warn mencatat pesan pada tingkat WARN dengan zero allocation
func (l *Logger) Warn(msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = WARN
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = WARN
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, nil)
	}
}

// Error logs a message at ERROR level with zero allocation
// Error mencatat pesan pada tingkat ERROR dengan zero allocation
func (l *Logger) Error(msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = ERROR
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = ERROR
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, nil)
	}
}

// Fatal logs a message at FATAL level and exits the program with zero allocation
// Fatal mencatat pesan pada tingkat FATAL dan keluar dari program dengan zero allocation
func (l *Logger) Fatal(msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = FATAL
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = FATAL
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, nil)
	}
}

// Panic logs a message at PANIC level and panics with zero allocation
// Panic mencatat pesan pada tingkat PANIC dan panic dengan zero allocation
func (l *Logger) Panic(msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = PANIC
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = PANIC
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, nil)
	}
}

// TraceContext logs a message at TRACE level with context support and zero allocation
// TraceContext mencatat pesan pada tingkat TRACE dengan dukungan konteks dan zero allocation
func (l *Logger) TraceContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = TRACE
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = TRACE
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, ctx)
	}
}

// DebugContext logs a message at DEBUG level with context support and zero allocation
// DebugContext mencatat pesan pada tingkat DEBUG dengan dukungan konteks dan zero allocation
func (l *Logger) DebugContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = DEBUG
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = DEBUG
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, ctx)
	}
}

// InfoContext logs a message at INFO level with context support and zero allocation
// InfoContext mencatat pesan pada tingkat INFO dengan dukungan konteks dan zero allocation
func (l *Logger) InfoContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = INFO
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = INFO
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, ctx)
	}
}

// NoticeContext logs a message at NOTICE level with context support and zero allocation
// NoticeContext mencatat pesan pada tingkat NOTICE dengan dukungan konteks dan zero allocation
func (l *Logger) NoticeContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = NOTICE
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = NOTICE
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, ctx)
	}
}

// WarnContext logs a message at WARN level with context support and zero allocation
// WarnContext mencatat pesan pada tingkat WARN dengan dukungan konteks dan zero allocation
func (l *Logger) WarnContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = WARN
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = WARN
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, ctx)
	}
}

// ErrorContext logs a message at ERROR level with context support and zero allocation
// ErrorContext mencatat pesan pada tingkat ERROR dengan dukungan konteks dan zero allocation
func (l *Logger) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = ERROR
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = ERROR
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, ctx)
	}
}

// FatalContext logs a message at FATAL level with context support, exits the program, and zero allocation
// FatalContext mencatat pesan pada tingkat FATAL dengan dukungan konteks, keluar dari program, dan zero allocation
func (l *Logger) FatalContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = FATAL
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = FATAL
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, ctx)
	}
}

// PanicContext logs a message at PANIC level with context support, panics, and zero allocation
// PanicContext mencatat pesan pada tingkat PANIC dengan dukungan konteks, panic, dan zero allocation
func (l *Logger) PanicContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.asyncLogger != nil {
		// For async logging, we still need to create a map for now
		// For future optimization, we could pass the fields directly
		entry := getEntryFromPool()
		entry.Level = PANIC
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.asyncLogger.LogEntry(entry)
	} else {
		entry := getEntryFromPool()
		entry.Level = PANIC
		copy(entry.Message[:], msg)
		entry.MessageLen = len(msg)
		addFieldsToEntry(entry, fields...)
		l.logEntry(entry, ctx)
	}
}

// Helper function to convert variadic fields to map
// Fungsi bantuan untuk mengkonversi field variadic ke map
func addFieldsToEntry(entry *LogEntry, fields ...interface{}) {
	if len(fields) == 0 {
		return
	}
	
	// Must have even number of arguments (key-value pairs)
	// Harus memiliki jumlah argumen genap (pasangan key-value)
	if len(fields)%2 != 0 {
		entry.SetField("error", "invalid field format - must be key-value pairs")
		return
	}
	
	for i := 0; i < len(fields); i += 2 {
		// Convert key to string if it's not already
		// Konversi key ke string jika belum
		var key string
		switch k := fields[i].(type) {
		case string:
			key = k
		case fmt.Stringer:
			key = k.String()
		default:
			// For zero allocation, we'll use a simple approach
			// For more complex cases, we could use a buffer pool
			key = fmt.Sprintf("%v", k)
		}
		
		// Add field to entry using zero-allocation methods when possible
		// Tambahkan field ke entry menggunakan metode zero-allocation jika memungkinkan
		switch v := fields[i+1].(type) {
		case string:
			entry.SetStringField(key, v)
		case int:
			entry.SetIntField(key, v)
		case int64:
			entry.SetIntField(key, int(v))
		case float64:
			entry.SetFloat64Field(key, v)
		case bool:
			entry.SetBoolField(key, v)
		default:
			// For other types, use the generic SetField method
			// Untuk tipe lain, gunakan metode SetField generik
			entry.SetField(key, v)
		}
	}
}

// Helper function to find last index of byte in string for zero allocation
// Fungsi bantuan untuk menemukan indeks terakhir dari byte dalam string untuk zero allocation
func lastIndexByte(s string, c byte) int {
	// Convert string to bytes without allocation using unsafe operations
	// Konversi string ke byte tanpa alokasi menggunakan operasi unsafe
	b := sToBytes(s)
	// Search from end for better performance on file paths
	// Cari dari akhir untuk kinerja yang lebih baik pada path file
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == c {
			return i
		}
	}
	return -1
}

// Helper function to get hostname with fallback for zero allocation
// Fungsi bantuan untuk mendapatkan nama host dengan fallback untuk zero allocation
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// Default error handler that writes to stderr
// Handler kesalahan default yang menulis ke stderr
func defaultErrorHandler(err error) {
	fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
}

// Default fatal handler that writes to stderr
// Handler fatal default yang menulis ke stderr
func defaultFatalHandler(entry *LogEntry) {
	fmt.Fprintf(os.Stderr, "Fatal error occurred: %s\n", bToString(entry.Message[:entry.MessageLen]))
}

// Default panic handler that writes to stderr
// Handler panic default yang menulis ke stderr
func defaultPanicHandler(entry *LogEntry) {
	fmt.Fprintf(os.Stderr, "Panic occurred: %s\n", bToString(entry.Message[:entry.MessageLen]))
}

// LoggerStats holds statistics about logger performance for monitoring and optimization
// LoggerStats menyimpan statistik tentang kinerja logger untuk monitoring dan optimasi
type LoggerStats struct {
	LogCounts    map[Level]int64 // Count of logs by level - Jumlah log berdasarkan tingkat
	BytesWritten int64           // Total bytes written - Total byte yang ditulis
	StartTime    time.Time       // Logger start time - Waktu mulai logger
	mu          sync.RWMutex
}

// NewLoggerStats creates a new LoggerStats
func NewLoggerStats() *LoggerStats {
	return &LoggerStats{
		LogCounts:    make(map[Level]int64),
		BytesWritten: 0,
		StartTime:    time.Now(),
	}
}

// GetStats returns a copy of current logger statistics for monitoring
// GetStats mengembalikan salinan statistik logger saat ini untuk monitoring
func (l *Logger) GetStats() *LoggerStats {
	if l.stats == nil {
		return nil
	}
	
	// Create a copy to avoid race conditions
	// Buat salinan untuk menghindari kondisi race
	stats := &LoggerStats{
		LogCounts:    make(map[Level]int64),
		BytesWritten: l.stats.BytesWritten,
		StartTime:    l.stats.StartTime,
	}
	
	// Copy log counts with read lock
	// Salin jumlah log dengan kunci baca
	l.stats.mu.RLock()
	for level, count := range l.stats.LogCounts {
		stats.LogCounts[level] = count
	}
	l.stats.mu.RUnlock()
	
	return stats
}

// RotatingFileWriter handles log file rotation with zero-allocation design
// RotatingFileWriter menangani rotasi file log dengan desain zero-allocation
type RotatingFileWriter struct {
	filename   string
	maxSize    int64
	maxAge     time.Duration
	maxBackups int
	localTime  bool
	compress   bool
	
	file      *os.File
	size      int64
	mu        sync.Mutex
	millCh    chan bool
	startMill sync.Once
}

// SamplingLogger provides log sampling functionality to reduce output volume
// SamplingLogger menyediakan fungsionalitas sampling log untuk mengurangi volume output
type SamplingLogger struct {
	logger      *Logger
	rate        int
	count       int64
	sampleCount int64
	mu          sync.Mutex
}

// NewSamplingLogger creates a new SamplingLogger
func NewSamplingLogger(logger *Logger, rate int) *SamplingLogger {
	return &SamplingLogger{
		logger: logger,
		rate:   rate,
	}
}

// shouldLog determines if the current log entry should be output based on sampling rate
// shouldLog menentukan apakah entri log saat ini harus dioutput berdasarkan tingkat sampling
func (s *SamplingLogger) shouldLog() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.count++
	if s.count%int64(s.rate) == 0 {
		s.sampleCount++
		return true
	}
	return false
}

// AsyncLogger provides asynchronous logging for non-blocking operations
// AsyncLogger menyediakan logging asinkron untuk operasi non-blocking
type AsyncLogger struct {
	logger    *Logger
	jobs      chan *logJob
	workers   int
	wg        sync.WaitGroup
	once      sync.Once
	closeOnce sync.Once
	closed    chan struct{}
}

// logJob represents a logging job for async processing
// logJob merepresentasikan pekerjaan logging untuk pemrosesan async
type logJob struct {
	level   Level
	msg     string
	fields  map[string]interface{}
	ctx     context.Context
	entry   *LogEntry // For zero-allocation approach
}

// NewAsyncLogger creates a new AsyncLogger
func NewAsyncLogger(logger *Logger, workers int, bufferSize int) *AsyncLogger {
	if workers <= 0 {
		workers = 1
	}
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	
	al := &AsyncLogger{
		logger:  logger,
		jobs:    make(chan *logJob, bufferSize),
		workers: workers,
		closed:  make(chan struct{}),
	}
	
	// Start worker goroutines
	// Mulai goroutine worker
	for i := 0; i < workers; i++ {
		al.wg.Add(1)
		go al.worker()
	}
	
	return al
}

// worker processes log jobs from the channel
// worker memproses pekerjaan log dari channel
func (al *AsyncLogger) worker() {
	defer al.wg.Done()
	
	for {
		select {
		case job := <-al.jobs:
			// Process the log job synchronously
			// Proses pekerjaan log secara sinkron
			if job.entry != nil {
				// Use the zero-allocation approach
				al.logger.logEntry(job.entry, job.ctx)
			} else {
				// Use the legacy approach for backward compatibility
				al.logger.logEntry(createLogEntry(job.level, job.msg, job.fields), job.ctx)
			}
		case <-al.closed:
			// Drain remaining jobs before closing
			// Kosongkan pekerjaan yang tersisa sebelum menutup
			for {
				select {
				case job := <-al.jobs:
					if job.entry != nil {
						// Use the zero-allocation approach
						al.logger.logEntry(job.entry, job.ctx)
					} else {
						// Use the legacy approach for backward compatibility
						al.logger.log(job.level, job.msg, job.fields, job.ctx)
					}
				default:
					return
				}
			}
		}
	}
}

// Log adds a log job to the queue for async processing
// Log menambahkan pekerjaan log ke antrian untuk pemrosesan async
func (al *AsyncLogger) Log(level Level, msg string, fields map[string]interface{}, ctx context.Context) {
	job := &logJob{
		level:  level,
		msg:    msg,
		fields: fields,
		ctx:    ctx,
	}
	
	select {
	case al.jobs <- job:
		// Job queued successfully
		// Pekerjaan diantrekan dengan sukses
	default:
		// Queue is full, drop the log to prevent blocking
		// Antrian penuh, hapus log untuk mencegah blocking
		if al.logger.errorHandler != nil {
			al.logger.errorHandler(fmt.Errorf("async log queue full, dropping log entry"))
		}
	}
}

// LogEntry adds a LogEntry job to the queue for async processing with zero allocation
// LogEntry menambahkan pekerjaan LogEntry ke antrian untuk pemrosesan async dengan zero allocation
func (al *AsyncLogger) LogEntry(entry *LogEntry) {
	job := &logJob{
		entry: entry,
	}
	
	select {
	case al.jobs <- job:
		// Job queued successfully
		// Pekerjaan diantrekan dengan sukses
	default:
		// Queue is full, drop the log to prevent blocking
		// Antrian penuh, hapus log untuk mencegah blocking
		// Return the entry to the pool since we're dropping it
		putEntryToPool(entry)
		if al.logger.errorHandler != nil {
			al.logger.errorHandler(fmt.Errorf("async log queue full, dropping log entry"))
		}
	}
}

// Close shuts down the async logger gracefully
// Close mematikan logger async dengan anggun
func (al *AsyncLogger) Close() {
	al.closeOnce.Do(func() {
		close(al.closed)
		close(al.jobs)
		al.wg.Wait()
	})
}

// BufferedWriter - ZERO ALLOCATION VERSION for high-performance buffered writing with minimal garbage collection
// BufferedWriter - VERSI ZERO ALLOCATION untuk penulisan buffer berkinerja tinggi dengan garbage collection minimal
type BufferedWriter struct {
	writer        io.Writer     // Underlying writer for output destination
	buffer        chan []byte   // Channel buffer for storing log entries awaiting write
	bufferSize    int           // Size of the channel buffer to control memory usage
	flushInterval time.Duration // Interval between automatic flush operations
	done          chan struct{} // Channel to signal worker shutdown
	wg            sync.WaitGroup // WaitGroup to coordinate worker shutdown
	droppedLogs   int64         // Counter for dropped logs when buffer is full
	totalLogs     int64         // Counter for total logs processed
	lastFlush     time.Time     // Timestamp of the last flush operation
	batchSize     int           // Maximum number of entries per batch write
	batchTimeout  time.Duration // Maximum time to wait for a full batch
}

// NewBufferedWriter creates a new BufferedWriter with specified configuration for high-performance buffered writing
// NewBufferedWriter membuat BufferedWriter baru dengan konfigurasi yang ditentukan untuk penulisan buffer berkinerja tinggi
func NewBufferedWriter(writer io.Writer, bufferSize int, flushInterval time.Duration) *BufferedWriter {
	bw := &BufferedWriter{
		writer:        writer,        // Set the underlying output writer
		buffer:        make(chan []byte, bufferSize), // Create buffered channel with specified size
		bufferSize:    bufferSize,    // Store buffer size for statistics
		flushInterval: flushInterval, // Set automatic flush interval
		done:          make(chan struct{}), // Initialize shutdown channel
		lastFlush:     time.Now(),    // Initialize last flush timestamp
		batchSize:     100,           // Set default batch size for efficient writes
		batchTimeout:  100 * time.Millisecond, // Set default batch timeout
	}
	
	// Start background flush worker goroutine
	// Mulai goroutine worker flush latar belakang
	bw.wg.Add(1)
	go bw.flushWorker()
	
	return bw
}

// Write adds data to the buffer for asynchronous writing with zero allocation
// Write menambahkan data ke buffer untuk penulisan asinkron dengan zero allocation
func (bw *BufferedWriter) Write(data []byte) error {
	// Increment total logs counter atomically for thread safety
	// Tambahkan penghitung total log secara atomik untuk keamanan thread
	atomic.AddInt64(&bw.totalLogs, 1)
	
	// Attempt to send data to buffer channel with non-blocking send
	// Coba kirim data ke channel buffer dengan pengiriman non-blocking
	select {
	case bw.buffer <- data:
		// Data successfully queued for writing
		// Data berhasil diantrekan untuk penulisan
		return nil
	default:
		// Buffer is full, increment dropped logs counter
		// Buffer penuh, tambahkan penghitung log yang dijatuhkan
		atomic.AddInt64(&bw.droppedLogs, 1)
		return fmt.Errorf("buffer full, log entry dropped")
	}
}

// flushWorker runs in background to periodically flush buffered data
// flushWorker berjalan di latar belakang untuk secara berkala mengosongkan data yang di-buffer
func (bw *BufferedWriter) flushWorker() {
	defer bw.wg.Done()
	
	// Create ticker for periodic flush based on configured interval
	// Buat ticker untuk flush berkala berdasarkan interval yang dikonfigurasi
	ticker := time.NewTicker(bw.flushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-bw.done:
			// Shutdown signal received, flush remaining data
			// Sinyal shutdown diterima, kosongkan data yang tersisa
			bw.flush()
			return
		case <-ticker.C:
			// Periodic flush interval reached
			// Interval flush berkala tercapai
			bw.flush()
		}
	}
}

// flush writes all buffered data to the underlying writer
// flush menulis semua data yang di-buffer ke penulis yang mendasarinya
func (bw *BufferedWriter) flush() {
	// Collect batch of entries to write efficiently
	// Kumpulkan batch entri untuk menulis secara efisien
	var batch [][]byte
	
	// Collect up to batchSize entries or until timeout
	// Kumpulkan hingga entri batchSize atau hingga timeout
	timeout := time.After(bw.batchTimeout)
	
collectLoop:
	for len(batch) < bw.batchSize {
		select {
		case data := <-bw.buffer:
			// Add entry to batch
			// Tambahkan entri ke batch
			batch = append(batch, data)
		case <-timeout:
			// Batch timeout reached
			// Timeout batch tercapai
			break collectLoop
		default:
			// No more data available
			// Tidak ada data lagi yang tersedia
			break collectLoop
		}
	}
	
	// Write batch if we have any data
	// Tulis batch jika kita memiliki data apa pun
	if len(batch) > 0 {
		// Concatenate all entries for single write operation to minimize system calls
		// Gabungkan semua entri untuk operasi tulis tunggal untuk meminimalkan panggilan sistem
		totalLen := 0
		for _, data := range batch {
			totalLen += len(data)
		}
		
		// Create combined buffer for efficient write
		// Buat buffer gabungan untuk tulis yang efisien
		combined := make([]byte, totalLen)
		offset := 0
		for _, data := range batch {
			copy(combined[offset:], data)
			offset += len(data)
		}
		
		// Write combined data to underlying writer
		// Tulis data gabungan ke penulis yang mendasarinya
		_, err := bw.writer.Write(combined)
		if err != nil {
			// Handle write error - in a real implementation, we would use the logger's error handler
			// Tangani kesalahan penulisan - dalam implementasi nyata, kita akan menggunakan handler kesalahan logger
			fmt.Printf("failed to flush buffered data: %v\n", err)
		}
		
		// Update last flush timestamp
		// Perbarui timestamp flush terakhir
		bw.lastFlush = time.Now()
	}
}

// Close shuts down the buffered writer and flushes remaining data
// Close mematikan penulis buffer dan mengosongkan data yang tersisa
func (bw *BufferedWriter) Close() error {
	// Signal worker to shutdown
	// Sinyalkan worker untuk shutdown
	close(bw.done)
	
	// Wait for worker to finish
	// Tunggu worker selesai
	bw.wg.Wait()
	
	// Final flush of any remaining data
	// Flush akhir dari data yang tersisa
	bw.flush()
	
	return nil
}

// ErrorHandler interface for handling logger errors
// Interface ErrorHandler untuk menangani kesalahan logger
type ErrorHandler interface {
	HandleError(error)
}



// DefaultMetricsCollector provides default implementation of MetricsCollector
// DefaultMetricsCollector menyediakan implementasi default dari MetricsCollector
type DefaultMetricsCollector struct {
	logCounts      map[Level]int64
	bytesWritten   int64
	mu             sync.RWMutex
}

// NewDefaultMetricsCollector creates a new DefaultMetricsCollector
func NewDefaultMetricsCollector() *DefaultMetricsCollector {
	return &DefaultMetricsCollector{
		logCounts: make(map[Level]int64),
	}
}

// IncrementLogCount increments the log count for the specified level
// IncrementLogCount menambah jumlah log untuk tingkat yang ditentukan
func (dmc *DefaultMetricsCollector) IncrementLogCount(level Level) {
	dmc.mu.Lock()
	defer dmc.mu.Unlock()
	
	dmc.logCounts[level]++
}

// AddBytesWritten adds the specified number of bytes to the total bytes written
// AddBytesWritten menambahkan jumlah byte yang ditentukan ke total byte yang ditulis
func (dmc *DefaultMetricsCollector) AddBytesWritten(bytes int64) {
	dmc.mu.Lock()
	defer dmc.mu.Unlock()
	
	dmc.bytesWritten += bytes
}

// GetLogCount returns the log count for the specified level
// GetLogCount mengembalikan jumlah log untuk tingkat yang ditentukan
func (dmc *DefaultMetricsCollector) GetLogCount(level Level) int64 {
	dmc.mu.RLock()
	defer dmc.mu.RUnlock()
	
	return dmc.logCounts[level]
}

// GetTotalBytesWritten returns the total number of bytes written
// GetTotalBytesWritten mengembalikan jumlah total byte yang ditulis
func (dmc *DefaultMetricsCollector) GetTotalBytesWritten() int64 {
	dmc.mu.RLock()
	defer dmc.mu.RUnlock()
	
	return dmc.bytesWritten
}

// createLogEntry creates a LogEntry from legacy parameters for backward compatibility
// createLogEntry membuat LogEntry dari parameter legacy untuk kompatibilitas mundur
func createLogEntry(level Level, msg string, fields map[string]interface{}) *LogEntry {
	entry := getEntryFromPool()
	entry.Level = level
	copy(entry.Message[:], msg)
	entry.MessageLen = len(msg)
	
	if fields != nil {
		for key, value := range fields {
			entry.SetField(key, value)
		}
	}
	
	return entry
}

// log is the core logging method with zero-allocation design (legacy)
// log adalah metode logging inti dengan desain zero-allocation (legacy)
func (l *Logger) log(level Level, msg string, fields map[string]interface{}, ctx context.Context) {
	entry := getEntryFromPool()
	entry.Level = level
	copy(entry.Message[:], msg)
	entry.MessageLen = len(msg)
	
	if fields != nil {
		for key, value := range fields {
			entry.SetField(key, value)
		}
	}
	
	l.logEntry(entry, ctx)
}
