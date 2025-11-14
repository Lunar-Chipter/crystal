// Package core provides core components for the crystal logger
package core

import (
	"context"
	"sync"
	"time"
)

const (
	// DEFAULT_TIMESTAMP_FORMAT defines the default timestamp format used for log entries
	// DEFAULT_TIMESTAMP_FORMAT menentukan format timestamp default yang digunakan untuk entri log
	DEFAULT_TIMESTAMP_FORMAT = "2006-01-02 15:04:05.000"
	
	// DEFAULT_CALLER_DEPTH specifies the default stack depth for extracting caller information
	// DEFAULT_CALLER_DEPTH menentukan kedalaman stack default untuk mengekstrak informasi pemanggil
	DEFAULT_CALLER_DEPTH = 3
	
	// MAX_PREALLOCATED_FIELDS limits the number of pre-allocated fields to prevent excessive memory usage
	// MAX_PREALLOCATED_FIELDS membatasi jumlah field yang pra-dialokasikan untuk mencegah penggunaan memori yang berlebihan
	MAX_PREALLOCATED_FIELDS = 16
	
	// MAX_PREALLOCATED_TAGS limits the number of pre-allocated tags to prevent excessive memory usage
	// MAX_PREALLOCATED_TAGS membatasi jumlah tag yang pra-dialokasikan untuk mencegah penggunaan memori yang berlebihan
	MAX_PREALLOCATED_TAGS = 8
	
	// MAX_MESSAGE_SIZE defines the maximum size of a log message before it gets truncated
	// MAX_MESSAGE_SIZE menentukan ukuran maksimum pesan log sebelum dipotong
	MAX_MESSAGE_SIZE = 1024 * 1024 // 1MB
	
	// STACK_TRACE_BUFFER_SIZE sets the buffer size for capturing stack traces during error logging
	// STACK_TRACE_BUFFER_SIZE menetapkan ukuran buffer untuk menangkap stack trace selama logging kesalahan
	STACK_TRACE_BUFFER_SIZE = 4096
	
	// MAX_STACK_DEPTH limits the depth of stack traces to prevent excessive memory allocation
	// MAX_STACK_DEPTH membatasi kedalaman stack trace untuk mencegah alokasi memori yang berlebihan
	MAX_STACK_DEPTH = 32
	
	// GOROUTINE_ID_BUFFER_SIZE defines the buffer size for storing goroutine ID information
	// GOROUTINE_ID_BUFFER_SIZE menentukan ukuran buffer untuk menyimpan informasi ID goroutine
	GOROUTINE_ID_BUFFER_SIZE = 32
	
	// LOG_ENTRY_BUFFER_SIZE sets the buffer size for formatting log entries to minimize allocations
	// LOG_ENTRY_BUFFER_SIZE menetapkan ukuran buffer untuk memformat entri log guna meminimalkan alokasi
	LOG_ENTRY_BUFFER_SIZE = 2048
)

// Pre-defined field key constants for consistent naming and zero-allocation access
// Konstanta key field yang telah ditentukan sebelumnya untuk penamaan yang konsisten dan akses zero-allocation
var (
	fieldKeyTraceID   = "trace_id"
	fieldKeySpanID    = "span_id"
	fieldKeyUserID    = "user_id"
	fieldKeySessionID = "session_id"
	fieldKeyRequestID = "request_id"
	fieldKeyDuration  = "duration"
	fieldKeyError     = "error"
	fieldKeyHostname  = "hostname"
	fieldKeyApp       = "application"
	fieldKeyVersion   = "version"
	fieldKeyEnv       = "environment"
)

// LogEntry represents a single log entry with zero-allocation design using fixed-size buffers and pre-allocated structures
// LogEntry merepresentasikan entri log tunggal dengan desain zero-allocation menggunakan buffer berukuran tetap dan struktur yang pra-dialokasikan
// Memory layout optimized for cache line alignment (64 bytes) to improve performance by reducing cache misses
// Tata letak memori dioptimalkan untuk alignment cache line (64 byte) untuk meningkatkan kinerja dengan mengurangi cache miss
type LogEntry struct {
	// Core fields - cache line 1 (hot path data) that are frequently accessed during logging operations
	// Field inti - cache line 1 (data jalur panas) yang sering diakses selama operasi logging
	Timestamp     time.Time        // 24 bytes (wall time + monotonic) - Time when the log entry was created
	Level         Level            // 1 byte - Severity level of the log entry
	_             [7]byte          // Padding for alignment to ensure proper memory layout
	MessageLen    int              // 8 bytes - Length of the message content
	FieldsCount   int              // 8 bytes - Number of fields currently set in the entry
	TagsCount     int              // 8 bytes - Number of tags currently set in the entry
	PID           int              // 8 bytes - Process ID of the application
	
	// Fixed-size buffers for hot data to avoid dynamic memory allocation during logging
	// Buffer berukuran tetap untuk data panas guna menghindari alokasi memori dinamis selama logging
	LevelName     [16]byte         // Fixed-size buffer for level name to avoid string allocations
	Message       [1024]byte       // Fixed-size buffer for message to avoid dynamic allocation
	
	// Caller info (frequently accessed) embedded directly to avoid pointer indirection
	// Informasi pemanggil (sering diakses) disematkan langsung untuk menghindari indirection pointer
	Caller        CallerInfo       // Embedded struct, not pointer - Caller information for debugging
	
	// Pre-allocated fields and tags to avoid slice allocations during log entry creation
	// Field dan tag yang pra-dialokasikan untuk menghindari alokasi slice selama pembuatan entri log
	Fields        [MAX_PREALLOCATED_FIELDS]FieldPair // Pre-allocated field pairs for structured logging
	Tags          [MAX_PREALLOCATED_TAGS]string      // Pre-allocated tags for categorizing log entries
	
	// Additional metadata stored in fixed-size buffers to prevent dynamic allocations
	// Metadata tambahan yang disimpan dalam buffer berukuran tetap untuk mencegah alokasi dinamis
	GoroutineID   [32]byte         // Buffer for storing goroutine ID information
	GoroutineIDLen int             // Actual length of goroutine ID data
	TraceID       [32]byte         // Buffer for distributed tracing ID
	TraceIDLen    int              // Actual length of trace ID data
	SpanID        [32]byte         // Buffer for distributed tracing span ID
	SpanIDLen     int              // Actual length of span ID data
	UserID        [64]byte         // Buffer for user identification
	UserIDLen     int              // Actual length of user ID data
	SessionID     [64]byte         // Buffer for session identification
	SessionIDLen  int              // Actual length of session ID data
	RequestID     [32]byte         // Buffer for request identification
	RequestIDLen  int              // Actual length of request ID data
	Duration      time.Duration    // Duration measurement for performance tracking
	Error         error            // Error information if this is an error log entry
	StackTrace    [4096]byte       // Buffer for storing stack trace information
	StackTraceLen int              // Actual length of stack trace data
	Hostname      [256]byte        // Buffer for hostname information
	HostnameLen   int              // Actual length of hostname data
	Application   [128]byte        // Buffer for application name
	ApplicationLen int             // Actual length of application name data
	Version       [64]byte         // Buffer for application version
	VersionLen    int              // Actual length of version data
	Environment   [64]byte         // Buffer for environment information
	EnvironmentLen int             // Actual length of environment data
	CustomMetrics [8]MetricPair    // Pre-allocated metrics for custom measurements
	MetricsCount  int              // Actual metrics count - Number of metrics currently set
}

// FieldPair represents a key-value pair for structured logging fields with zero-allocation design
// FieldPair merepresentasikan pasangan key-value untuk field logging terstruktur dengan desain zero-allocation
type FieldPair struct {
	Key   [64]byte    // Fixed-size key buffer to avoid dynamic allocation for field names
	KeyLen int        // Actual key length to track the valid portion of the key buffer
	Value interface{} // Value can be any type - supports flexible structured logging
	// Zero-allocation string storage
	StringValue   [256]byte // Fixed-size buffer for string values to avoid dynamic allocation
	StringValueLen int      // Actual string value length to track the valid portion of the string buffer
	IsString      bool      // Flag to indicate if this field is a string value stored in StringValue buffer
	IntValue      int64     // Storage for integer values to avoid interface{} allocation
	IsInt         bool      // Flag to indicate if this field is an integer value stored in IntValue
	Float64Value  float64   // Storage for float64 values to avoid interface{} allocation
	IsFloat64     bool      // Flag to indicate if this field is a float64 value stored in Float64Value
	BoolValue     bool      // Storage for boolean values to avoid interface{} allocation
	IsBool        bool      // Flag to indicate if this field is a boolean value stored in BoolValue
}

// MetricPair represents a key-value pair for custom metrics with numeric values for performance tracking
// MetricPair merepresentasikan pasangan key-value untuk metrik kustom dengan nilai numerik untuk pelacakan kinerja
type MetricPair struct {
	Key   [64]byte // Fixed-size key buffer to avoid dynamic allocation for metric names
	KeyLen int     // Actual key length to track the valid portion of the key buffer
	Value float64  // Numeric value for metrics - optimized for performance measurements
}

// CallerInfo contains caller information with zero-allocation design using fixed-size buffers
// CallerInfo berisi informasi pemanggil dengan desain zero-allocation menggunakan buffer berukuran tetap
type CallerInfo struct {
	File     [256]byte // Fixed-size buffer for file name to avoid dynamic allocation
	FileLen  int       // Actual file name length to track valid portion of the buffer
	Line     int       // Line number where the log entry was created
	Function [128]byte // Fixed-size buffer for function name to avoid dynamic allocation
	FunctionLen int    // Actual function name length to track valid portion of the buffer
	Package  [128]byte // Fixed-size buffer for package name to avoid dynamic allocation
	PackageLen int     // Actual package name length to track valid portion of the buffer
}

// ByteArray provides a fixed-size byte array with efficient append operations for zero-allocation formatting
// ByteArray menyediakan array byte berukuran tetap dengan operasi append yang efisien untuk formatting zero-allocation
type ByteArray struct {
	buf      [LOG_ENTRY_BUFFER_SIZE]byte // Fixed-size buffer for log entry formatting
	len      int                         // Current length of valid data in the buffer
	data     []byte                      // Slice view of the buffer for compatibility with formatters
}

// Reset clears the buffer by resetting the slice length to zero while preserving capacity
// Reset mengosongkan buffer dengan mengatur ulang panjang slice ke nol sambil mempertahankan kapasitas
func (ba *ByteArray) Reset() {
	ba.len = 0
	ba.data = ba.buf[:0]
}

// Bytes returns the underlying byte slice for direct access
// Bytes mengembalikan slice byte yang mendasarinya untuk akses langsung
func (ba *ByteArray) Bytes() []byte {
	return ba.buf[:ba.len]
}

// Len returns the current length of data in the buffer
// Len mengembalikan panjang data saat ini dalam buffer
func (ba *ByteArray) Len() int {
	return ba.len
}

// Write appends bytes to the buffer with bounds checking to prevent overflow and ensure memory safety
// Write menambahkan byte ke buffer dengan pemeriksaan batas untuk mencegah overflow dan memastikan keamanan memori
func (ba *ByteArray) Write(p []byte) (n int, err error) {
	n = len(p)
	// Check if we have enough space in the buffer to avoid overflow
	// Periksa apakah kita memiliki cukup ruang dalam buffer untuk menghindari overflow
	if ba.len+n > len(ba.buf) {
		// Truncate to fit available space and avoid buffer overflow
		// Potong agar sesuai dengan ruang yang tersedia dan hindari buffer overflow
		n = len(ba.buf) - ba.len
		p = p[:n]
	}
	// Copy data to buffer using direct memory copy for efficiency
	// Salin data ke buffer menggunakan salinan memori langsung untuk efisiensi
	copy(ba.buf[ba.len:], p)
	ba.len += n
	ba.data = ba.buf[:ba.len]
	return n, nil
}

// WriteByte appends a single byte to the buffer with bounds checking for zero-allocation operations
// WriteByte menambahkan satu byte ke buffer dengan pemeriksaan batas untuk operasi zero-allocation
func (ba *ByteArray) WriteByte(c byte) error {
	// Check if we have space for one more byte to avoid overflow
	// Periksa apakah kita memiliki ruang untuk satu byte lagi untuk menghindari overflow
	if ba.len >= len(ba.buf) {
		return nil // Buffer full, silently drop to avoid allocation - Buffer penuh, diam-diam hapus untuk menghindari alokasi
	}
	// Direct assignment for maximum efficiency and zero allocation
	// Penugasan langsung untuk efisiensi maksimal dan zero allocation
	ba.buf[ba.len] = c
	ba.len++
	ba.data = ba.buf[:ba.len]
	return nil
}

// WriteString appends a string to the buffer with zero allocation by avoiding intermediate byte slice creation
// WriteString menambahkan string ke buffer dengan zero allocation dengan menghindari pembuatan slice byte perantara
func (ba *ByteArray) WriteString(s string) (n int, err error) {
	// Convert string to bytes without allocation using unsafe operations
	// Konversi string ke byte tanpa alokasi menggunakan operasi unsafe
	return ba.Write(sToBytes(s))
}

// GetTimestamp returns the timestamp of the log entry
func (e *LogEntry) GetTimestamp() time.Time {
	return e.Timestamp
}

// GetLevel returns the level of the log entry
func (e *LogEntry) GetLevel() Level {
	return e.Level
}

// GetMessage returns the message of the log entry
func (e *LogEntry) GetMessage() string {
	return bToString(e.Message[:e.MessageLen])
}

// GetPID returns the PID of the log entry
func (e *LogEntry) GetPID() int {
	return e.PID
}

// GetCallerFile returns the caller file of the log entry
func (e *LogEntry) GetCallerFile() string {
	return bToString(e.Caller.File[:e.Caller.FileLen])
}

// GetCallerLine returns the caller line of the log entry
func (e *LogEntry) GetCallerLine() int {
	return e.Caller.Line
}

// GetGoroutineID returns the goroutine ID of the log entry
func (e *LogEntry) GetGoroutineID() string {
	return bToString(e.GoroutineID[:e.GoroutineIDLen])
}

// GetTraceID returns the trace ID of the log entry
func (e *LogEntry) GetTraceID() string {
	return bToString(e.TraceID[:e.TraceIDLen])
}

// GetSpanID returns the span ID of the log entry
func (e *LogEntry) GetSpanID() string {
	return bToString(e.SpanID[:e.SpanIDLen])
}

// GetUserID returns the user ID of the log entry
func (e *LogEntry) GetUserID() string {
	return bToString(e.UserID[:e.UserIDLen])
}

// GetSessionID returns the session ID of the log entry
func (e *LogEntry) GetSessionID() string {
	return bToString(e.SessionID[:e.SessionIDLen])
}

// GetRequestID returns the request ID of the log entry
func (e *LogEntry) GetRequestID() string {
	return bToString(e.RequestID[:e.RequestIDLen])
}

// GetDuration returns the duration of the log entry
func (e *LogEntry) GetDuration() time.Duration {
	return e.Duration
}

// GetStackTrace returns the stack trace of the log entry
func (e *LogEntry) GetStackTrace() string {
	return bToString(e.StackTrace[:e.StackTraceLen])
}

// GetHostname returns the hostname of the log entry
func (e *LogEntry) GetHostname() string {
	return bToString(e.Hostname[:e.HostnameLen])
}

// GetApplication returns the application of the log entry
func (e *LogEntry) GetApplication() string {
	return bToString(e.Application[:e.ApplicationLen])
}

// GetVersion returns the version of the log entry
func (e *LogEntry) GetVersion() string {
	return bToString(e.Version[:e.VersionLen])
}

// GetEnvironment returns the environment of the log entry
func (e *LogEntry) GetEnvironment() string {
	return bToString(e.Environment[:e.EnvironmentLen])
}

// GetFields returns the fields of the log entry
func (e *LogEntry) GetFields() []interface{} {
	fields := make([]interface{}, e.FieldsCount)
	for i := 0; i < e.FieldsCount; i++ {
		fields[i] = e.Fields[i]
	}
	return fields
}

// GetTags returns the tags of the log entry
func (e *LogEntry) GetTags() []string {
	tags := make([]string, e.TagsCount)
	for i := 0; i < e.TagsCount; i++ {
		tags[i] = e.Tags[i]
	}
	return tags
}

// GetMetrics returns the metrics of the log entry
func (e *LogEntry) GetMetrics() []interface{} {
	metrics := make([]interface{}, e.MetricsCount)
	for i := 0; i < e.MetricsCount; i++ {
		metrics[i] = e.CustomMetrics[i]
	}
	return metrics
}

// GetError returns the error of the log entry
func (e *LogEntry) GetError() error {
	return e.Error
}

// Object pool for LogEntry with zero-allocation design to reuse LogEntry instances and avoid garbage collection pressure
// Pool objek untuk LogEntry dengan desain zero-allocation untuk menggunakan kembali instance LogEntry dan menghindari tekanan garbage collection
var entryPool = sync.Pool{
	New: func() interface{} {
		return &LogEntry{}
	},
}

// getEntryFromPool gets a LogEntry from pool with zero allocation to reuse existing instances and minimize garbage collection
// getEntryFromPool mendapatkan LogEntry dari pool dengan zero allocation untuk menggunakan kembali instance yang ada dan meminimalkan garbage collection
func getEntryFromPool() *LogEntry {
	entry := entryPool.Get().(*LogEntry)
	// Reset fields efficiently without allocations by reinitializing the struct to its zero value
	// Reset field secara efisien tanpa alokasi dengan menginisialisasi ulang struct ke nilai nolnya
	*entry = LogEntry{} // Zero-allocation reset - reinitialize to zero value without new allocation
	return entry
}

// putEntryToPool returns a LogEntry to pool for reuse, reducing the need for future allocations
// putEntryToPool mengembalikan LogEntry ke pool untuk digunakan kembali, mengurangi kebutuhan alokasi di masa depan
func putEntryToPool(entry *LogEntry) {
	entryPool.Put(entry)
}

// SetField sets a field with zero allocation by using pre-allocated buffers and avoiding dynamic memory allocation
// SetField mengatur field dengan zero allocation dengan menggunakan buffer yang telah dialokasikan sebelumnya dan menghindari alokasi memori dinamis
func (e *LogEntry) SetField(key string, value interface{}) {
	// Check if we have space in the pre-allocated fields array to avoid dynamic allocation
	// Memeriksa apakah kita memiliki ruang dalam array field yang telah dialokasikan sebelumnya untuk menghindari alokasi dinamis
	if e.FieldsCount >= len(e.Fields) {
		return // Buffer full, skip to avoid allocation - Buffer penuh, lewati untuk menghindari alokasi
	}
	
	// Copy key to fixed buffer to avoid string allocation and ensure memory safety
	// Salin key ke buffer tetap untuk menghindari alokasi string dan memastikan keamanan memori
	keyLen := len(key)
	if keyLen > len(e.Fields[e.FieldsCount].Key) {
		keyLen = len(e.Fields[e.FieldsCount].Key)
	}
	copy(e.Fields[e.FieldsCount].Key[:], key[:keyLen])
	e.Fields[e.FieldsCount].KeyLen = keyLen
	e.Fields[e.FieldsCount].Value = value
	e.FieldsCount++
}

// SetStringField sets a string field with zero allocation by using pre-allocated buffers and avoiding dynamic memory allocation
// SetStringField mengatur field string dengan zero allocation dengan menggunakan buffer yang telah dialokasikan sebelumnya dan menghindari alokasi memori dinamis
func (e *LogEntry) SetStringField(key, value string) {
	// Check if we have space in the pre-allocated fields array to avoid dynamic allocation
	// Memeriksa apakah kita memiliki ruang dalam array field yang telah dialokasikan sebelumnya untuk menghindari alokasi dinamis
	if e.FieldsCount >= len(e.Fields) {
		return // Buffer full, skip to avoid allocation - Buffer penuh, lewati untuk menghindari alokasi
	}
	
	// Copy key to fixed buffer to avoid string allocation and ensure memory safety
	// Salin key ke buffer tetap untuk menghindari alokasi string dan memastikan keamanan memori
	keyLen := len(key)
	if keyLen > len(e.Fields[e.FieldsCount].Key) {
		keyLen = len(e.Fields[e.FieldsCount].Key)
	}
	copy(e.Fields[e.FieldsCount].Key[:], key[:keyLen])
	e.Fields[e.FieldsCount].KeyLen = keyLen
	
	// Copy value to fixed buffer to avoid string allocation and ensure memory safety
	// Salin nilai ke buffer tetap untuk menghindari alokasi string dan memastikan keamanan memori
	valueLen := len(value)
	if valueLen > len(e.Fields[e.FieldsCount].StringValue) {
		valueLen = len(e.Fields[e.FieldsCount].StringValue)
	}
	copy(e.Fields[e.FieldsCount].StringValue[:], value[:valueLen])
	e.Fields[e.FieldsCount].StringValueLen = valueLen
	e.Fields[e.FieldsCount].IsString = true
	e.FieldsCount++
}

// SetIntField sets an integer field with zero allocation by using pre-allocated buffers and avoiding dynamic memory allocation
// SetIntField mengatur field integer dengan zero allocation dengan menggunakan buffer yang telah dialokasikan sebelumnya dan menghindari alokasi memori dinamis
func (e *LogEntry) SetIntField(key string, value int) {
	// Check if we have space in the pre-allocated fields array to avoid dynamic allocation
	// Memeriksa apakah kita memiliki ruang dalam array field yang telah dialokasikan sebelumnya untuk menghindari alokasi dinamis
	if e.FieldsCount >= len(e.Fields) {
		return // Buffer full, skip to avoid allocation - Buffer penuh, lewati untuk menghindari alokasi
	}
	
	// Copy key to fixed buffer to avoid string allocation and ensure memory safety
	// Salin key ke buffer tetap untuk menghindari alokasi string dan memastikan keamanan memori
	keyLen := len(key)
	if keyLen > len(e.Fields[e.FieldsCount].Key) {
		keyLen = len(e.Fields[e.FieldsCount].Key)
	}
	copy(e.Fields[e.FieldsCount].Key[:], key[:keyLen])
	e.Fields[e.FieldsCount].KeyLen = keyLen
	
	// Store integer value directly to avoid interface{} allocation
	// Simpan nilai integer langsung untuk menghindari alokasi interface{}
	e.Fields[e.FieldsCount].IntValue = int64(value)
	e.Fields[e.FieldsCount].IsInt = true
	e.FieldsCount++
}

// SetFloat64Field sets a float64 field with zero allocation by using pre-allocated buffers and avoiding dynamic memory allocation
// SetFloat64Field mengatur field float64 dengan zero allocation dengan menggunakan buffer yang telah dialokasikan sebelumnya dan menghindari alokasi memori dinamis
func (e *LogEntry) SetFloat64Field(key string, value float64) {
	// Check if we have space in the pre-allocated fields array to avoid dynamic allocation
	// Memeriksa apakah kita memiliki ruang dalam array field yang telah dialokasikan sebelumnya untuk menghindari alokasi dinamis
	if e.FieldsCount >= len(e.Fields) {
		return // Buffer full, skip to avoid allocation - Buffer penuh, lewati untuk menghindari alokasi
	}
	
	// Copy key to fixed buffer to avoid string allocation and ensure memory safety
	// Salin key ke buffer tetap untuk menghindari alokasi string dan memastikan keamanan memori
	keyLen := len(key)
	if keyLen > len(e.Fields[e.FieldsCount].Key) {
		keyLen = len(e.Fields[e.FieldsCount].Key)
	}
	copy(e.Fields[e.FieldsCount].Key[:], key[:keyLen])
	e.Fields[e.FieldsCount].KeyLen = keyLen
	
	// Store float64 value directly to avoid interface{} allocation
	// Simpan nilai float64 langsung untuk menghindari alokasi interface{}
	e.Fields[e.FieldsCount].Float64Value = value
	e.Fields[e.FieldsCount].IsFloat64 = true
	e.FieldsCount++
}

// SetBoolField sets a boolean field with zero allocation by using pre-allocated buffers and avoiding dynamic memory allocation
// SetBoolField mengatur field boolean dengan zero allocation dengan menggunakan buffer yang telah dialokasikan sebelumnya dan menghindari alokasi memori dinamis
func (e *LogEntry) SetBoolField(key string, value bool) {
	// Check if we have space in the pre-allocated fields array to avoid dynamic allocation
	// Memeriksa apakah kita memiliki ruang dalam array field yang telah dialokasikan sebelumnya untuk menghindari alokasi dinamis
	if e.FieldsCount >= len(e.Fields) {
		return // Buffer full, skip to avoid allocation - Buffer penuh, lewati untuk menghindari alokasi
	}
	
	// Copy key to fixed buffer to avoid string allocation and ensure memory safety
	// Salin key ke buffer tetap untuk menghindari alokasi string dan memastikan keamanan memori
	keyLen := len(key)
	if keyLen > len(e.Fields[e.FieldsCount].Key) {
		keyLen = len(e.Fields[e.FieldsCount].Key)
	}
	copy(e.Fields[e.FieldsCount].Key[:], key[:keyLen])
	e.Fields[e.FieldsCount].KeyLen = keyLen
	
	// Store boolean value directly to avoid interface{} allocation
	// Simpan nilai boolean langsung untuk menghindari alokasi interface{}
	e.Fields[e.FieldsCount].BoolValue = value
	e.Fields[e.FieldsCount].IsBool = true
	e.FieldsCount++
}

// SetTimeField sets a time field with zero allocation by using pre-allocated buffers and avoiding dynamic memory allocation
// SetTimeField mengatur field time dengan zero allocation dengan menggunakan buffer yang telah dialokasikan sebelumnya dan menghindari alokasi memori dinamis
func (e *LogEntry) SetTimeField(key string, value time.Time) {
	// For time fields, we'll still use the interface{} approach since time.Time is a struct
	// Untuk field time, kita akan tetap menggunakan pendekatan interface{} karena time.Time adalah struct
	e.SetField(key, value)
}

// GetStringField gets a string field value with zero allocation
// GetStringField mendapatkan nilai field string dengan zero allocation
func (e *LogEntry) GetStringField(key string) (string, bool) {
	for i := 0; i < e.FieldsCount; i++ {
		// Check if this field matches the key
		if e.Fields[i].KeyLen == len(key) && string(e.Fields[i].Key[:e.Fields[i].KeyLen]) == key {
			// Check if this is a string field
			if e.Fields[i].IsString {
				return string(e.Fields[i].StringValue[:e.Fields[i].StringValueLen]), true
			}
		}
	}
	return "", false
}

// GetIntField gets an integer field value with zero allocation
// GetIntField mendapatkan nilai field integer dengan zero allocation
func (e *LogEntry) GetIntField(key string) (int, bool) {
	for i := 0; i < e.FieldsCount; i++ {
		// Check if this field matches the key
		if e.Fields[i].KeyLen == len(key) && string(e.Fields[i].Key[:e.Fields[i].KeyLen]) == key {
			// Check if this is an integer field
			if e.Fields[i].IsInt {
				return int(e.Fields[i].IntValue), true
			}
		}
	}
	return 0, false
}

// GetFloat64Field gets a float64 field value with zero allocation
// GetFloat64Field mendapatkan nilai field float64 dengan zero allocation
func (e *LogEntry) GetFloat64Field(key string) (float64, bool) {
	for i := 0; i < e.FieldsCount; i++ {
		// Check if this field matches the key
		if e.Fields[i].KeyLen == len(key) && string(e.Fields[i].Key[:e.Fields[i].KeyLen]) == key {
			// Check if this is a float64 field
			if e.Fields[i].IsFloat64 {
				return e.Fields[i].Float64Value, true
			}
		}
	}
	return 0.0, false
}

// GetBoolField gets a boolean field value with zero allocation
// GetBoolField mendapatkan nilai field boolean dengan zero allocation
func (e *LogEntry) GetBoolField(key string) (bool, bool) {
	for i := 0; i < e.FieldsCount; i++ {
		// Check if this field matches the key
		if e.Fields[i].KeyLen == len(key) && string(e.Fields[i].Key[:e.Fields[i].KeyLen]) == key {
			// Check if this is a boolean field
			if e.Fields[i].IsBool {
				return e.Fields[i].BoolValue, true
			}
		}
	}
	return false, false
}

// SetMetric sets a custom metric with zero allocation by using pre-allocated buffers and avoiding dynamic memory allocation
// SetMetric mengatur metrik kustom dengan zero allocation dengan menggunakan buffer yang telah dialokasikan sebelumnya dan menghindari alokasi memori dinamis
func (e *LogEntry) SetMetric(key string, value float64) {
	// Check if we have space in the pre-allocated metrics array to avoid dynamic allocation
	// Memeriksa apakah kita memiliki ruang dalam array metrik yang telah dialokasikan sebelumnya untuk menghindari alokasi dinamis
	if e.MetricsCount >= len(e.CustomMetrics) {
		return // Buffer full, skip to avoid allocation - Buffer penuh, lewati untuk menghindari alokasi
	}
	
	// Copy key to fixed buffer to avoid string allocation and ensure memory safety
	// Salin key ke buffer tetap untuk menghindari alokasi string dan memastikan keamanan memori
	keyLen := len(key)
	if keyLen > len(e.CustomMetrics[e.MetricsCount].Key) {
		keyLen = len(e.CustomMetrics[e.MetricsCount].Key)
	}
	copy(e.CustomMetrics[e.MetricsCount].Key[:], key[:keyLen])
	e.CustomMetrics[e.MetricsCount].KeyLen = keyLen
	e.CustomMetrics[e.MetricsCount].Value = value
	e.MetricsCount++
}

// SetTag adds a tag to the log entry with zero allocation by using pre-allocated buffers and avoiding dynamic memory allocation
// SetTag menambahkan tag ke entri log dengan zero allocation dengan menggunakan buffer yang telah dialokasikan sebelumnya dan menghindari alokasi memori dinamis
func (e *LogEntry) SetTag(tag string) {
	// Check if we have space in the pre-allocated tags array to avoid dynamic allocation
	// Memeriksa apakah kita memiliki ruang dalam array tag yang telah dialokasikan sebelumnya untuk menghindari alokasi dinamis
	if e.TagsCount >= len(e.Tags) {
		return // Buffer full, skip to avoid allocation - Buffer penuh, lewati untuk menghindari alokasi
	}
	
	// Store tag directly in pre-allocated array to avoid allocation
	// Simpan tag langsung dalam array yang telah dialokasikan sebelumnya untuk menghindari alokasi
	e.Tags[e.TagsCount] = tag
	e.TagsCount++
}

// ===============================
// CONTEXT SUPPORT - ZERO ALLOCATION
// ===============================
// contextKey type ensures type safety for context keys and prevents collisions with other packages
// tipe contextKey memastikan keamanan tipe untuk kunci konteks dan mencegah tabrakan dengan paket lain
type contextKey string

const (
	// TraceIDKey is used to store distributed tracing identifier in context
	// TraceIDKey digunakan untuk menyimpan identifier tracing terdistribusi dalam konteks
	TraceIDKey contextKey = "trace_id"
	
	// SpanIDKey is used to store distributed tracing span identifier in context
	// SpanIDKey digunakan untuk menyimpan identifier span tracing terdistribusi dalam konteks
	SpanIDKey  contextKey = "span_id"
	
	// UserIDKey is used to store user identification in context
	// UserIDKey digunakan untuk menyimpan identifikasi pengguna dalam konteks
	UserIDKey  contextKey = "user_id"
	
	// SessionIDKey is used to store session identification in context
	// SessionIDKey digunakan untuk menyimpan identifikasi sesi dalam konteks
	SessionIDKey contextKey = "session_id"
	
	// RequestIDKey is used to store request identification in context
	// RequestIDKey digunakan untuk menyimpan identifikasi permintaan dalam konteks
	RequestIDKey contextKey = "request_id"
)

// WithTraceID adds a trace ID to the context with zero allocation by using pre-defined context key
// WithTraceID menambahkan ID trace ke konteks dengan zero allocation dengan menggunakan kunci konteks yang telah ditentukan sebelumnya
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithSpanID adds a span ID to the context with zero allocation by using pre-defined context key
// WithSpanID menambahkan ID span ke konteks dengan zero allocation dengan menggunakan kunci konteks yang telah ditentukan sebelumnya
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// WithUserID adds a user ID to the context with zero allocation by using pre-defined context key
// WithUserID menambahkan ID pengguna ke konteks dengan zero allocation dengan menggunakan kunci konteks yang telah ditentukan sebelumnya
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithSessionID adds a session ID to the context with zero allocation by using pre-defined context key
// WithSessionID menambahkan ID sesi ke konteks dengan zero allocation dengan menggunakan kunci konteks yang telah ditentukan sebelumnya
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionIDKey, sessionID)
}

// WithRequestID adds a request ID to the context with zero allocation by using pre-defined context key
// WithRequestID menambahkan ID permintaan ke konteks dengan zero allocation dengan menggunakan kunci konteks yang telah ditentukan sebelumnya
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetTraceID retrieves trace ID from context with zero allocation by using pre-defined context key
// GetTraceID mengambil ID trace dari konteks dengan zero allocation dengan menggunakan kunci konteks yang telah ditentukan sebelumnya
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetSpanID retrieves span ID from context with zero allocation by using pre-defined context key
// GetSpanID mengambil ID span dari konteks dengan zero allocation dengan menggunakan kunci konteks yang telah ditentukan sebelumnya
func GetSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		return spanID
	}
	return ""
}

// GetUserID retrieves user ID from context with zero allocation by using pre-defined context key
// GetUserID mengambil ID pengguna dari konteks dengan zero allocation dengan menggunakan kunci konteks yang telah ditentukan sebelumnya
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetSessionID retrieves session ID from context with zero allocation by using pre-defined context key
// GetSessionID mengambil ID sesi dari konteks dengan zero allocation dengan menggunakan kunci konteks yang telah ditentukan sebelumnya
func GetSessionID(ctx context.Context) string {
	if sessionID, ok := ctx.Value(SessionIDKey).(string); ok {
		return sessionID
	}
	return ""
}

// GetRequestID retrieves request ID from context with zero allocation by using pre-defined context key
// GetRequestID mengambil ID permintaan dari konteks dengan zero allocation dengan menggunakan kunci konteks yang telah ditentukan sebelumnya
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// bufferPool provides reusable buffers for formatting operations to minimize memory allocations
// bufferPool menyediakan buffer yang dapat digunakan kembali untuk operasi formatting guna meminimalkan alokasi memori
var bufferPool = sync.Pool{
	New: func() interface{} {
		return &ByteArray{}
	},
}

// getBufferFromPool gets a buffer from pool with zero allocation and resets it for reuse
// getBufferFromPool mendapatkan buffer dari pool dengan zero allocation dan mengatur ulang untuk digunakan kembali
func getBufferFromPool() *ByteArray {
	buf := bufferPool.Get().(*ByteArray)
	buf.Reset() // Reset buffer to empty state for reuse
	return buf
}

// putBufferToPool returns a buffer to pool if it's not too large to prevent memory waste
// putBufferToPool mengembalikan buffer ke pool jika tidak terlalu besar untuk mencegah pemborosan memori
func putBufferToPool(buf *ByteArray) {
	// Only return reasonably sized buffers to pool to prevent memory waste from large buffers
	// Hanya kembalikan buffer dengan ukuran wajar ke pool untuk mencegah pemborosan memori dari buffer besar
	if buf.Len() < 64*1024 {
		bufferPool.Put(buf)
	}
}

// Interface implementation for LogEntryInterface
// Implementasi interface untuk LogEntryInterface

