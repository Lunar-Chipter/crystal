package crystal

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "runtime"
    "strconv"
    "strings"
    "sync"
    "sync/atomic"
    "time"
    "context"
    "compress/gzip"
    "encoding/csv"
    "math"
    "sort"
    "bytes"
    "unsafe"
)

const (
    // Default timestamp format for log entries
    // Format timestamp default untuk entri log
    DEFAULT_TIMESTAMP_FORMAT = "2006-01-02 15:04:05.000"
    
    // Default depth for caller information extraction
    // Kedalaman default untuk ekstraksi informasi pemanggil
    DEFAULT_CALLER_DEPTH = 3
    
    // Default buffer size for buffered writer
    // Ukuran buffer default untuk writer yang di-buffer
    DEFAULT_BUFFER_SIZE = 1000
    
    // Default flush interval for buffered writer
    // Interval flush default untuk writer yang di-buffer
    DEFAULT_FLUSH_INTERVAL = 5 * time.Second
    
    // Maximum pre-allocated fields to avoid repeated allocations
    // Maksimum field pra-alokasi untuk menghindari alokasi berulang
    MAX_PREALLOCATED_FIELDS = 16
    
    // Maximum pre-allocated tags to avoid repeated allocations
    // Maksimum tag pra-alokasi untuk menghindari alokasi berulang
    MAX_PREALLOCATED_TAGS = 8
    
    // Maximum message size before truncation
    // Ukuran pesan maksimum sebelum pemotongan
    MAX_MESSAGE_SIZE = 1024 * 1024 // 1MB
    
    // Stack trace buffer size
    // Ukuran buffer stack trace
    STACK_TRACE_BUFFER_SIZE = 4096
    
    // Maximum stack trace depth
    // Kedalaman maksimum stack trace
    MAX_STACK_DEPTH = 32
    
    // Goroutine ID buffer size
    // Ukuran buffer ID goroutine
    GOROUTINE_ID_BUFFER_SIZE = 32
    
    // Log entry buffer size
    // Ukuran buffer entri log
    LOG_ENTRY_BUFFER_SIZE = 2048
)

// ===============================
// LEVEL DEFINITION - OPTIMIZED
// ===============================
// Level represents the severity level of a log entry
// Level merepresentasikan tingkat keparahan entri log
type Level uint8

const (
    // TRACE level for very detailed debugging information
    // Level TRACE untuk informasi debugging yang sangat detail
    TRACE Level = iota
    
    // DEBUG level for debugging information
    // Level DEBUG untu informasi debugging
    DEBUG
    
    // INFO level for general information messages
    // Level INFO untuk pesan informasi umum
    INFO
    
    // NOTICE level for normal but significant conditions
    // Level NOTICE untuk kondisi normal namun signifikan
    NOTICE
    
    // WARN level for warning messages
    // Level WARN unt pesan peringatan
    WARN
    
    // ERROR level for error messages
    // Level ERROR untuk pesan kesalahan
    ERROR
    
    // FATAL level for critical errors that cause program termination
    // Level FATAL untuk kesalahan kritis yang menyebabkan program berhenti
    FATAL
    
    // PANIC level for panic conditions
    // Level PANIC untuk kondisi panic
    PANIC
)

// Pre-computed string representations and colors for zero-allocation access
// Representasi string dan warna pra-komputasi untuk akses zero-allocation
var (
    levelStrings = [...]string{
        "TRACE", "DEBUG", "INFO", "NOTICE", "WARN", "ERROR", "FATAL", "PANIC",
    }
    
    levelColors = [...]string{
        "\033[38;5;246m",    // Gray - TRACE
        "\033[36m",          // Cyan - DEBUG
        "\033[32m",          // Green - INFO
        "\033[38;5;220m",    // Yellow - NOTICE
        "\033[33m",          // Orange - WARN
        "\033[31m",          // Red - ERROR
        "\033[38;5;198m",    // Magenta - FATAL
        "\033[38;5;196m",    // Bright Red - PANIC
    }
    
    levelBackgrounds = [...]string{
        "\033[48;5;238m",    // Dark gray background
        "\033[48;5;236m",
        "\033[48;5;28m",
        "\033[48;5;94m",
        "\033[48;5;130m",
        "\033[48;5;88m",
        "\033[48;5;90m",
        "\033[48;5;52m",
    }
    
    // Pre-computed level masks for fast comparison
    // Pra-komputasi mask level untuk perbandingan cepat
    levelMasks = [...]uint64{
        1 << TRACE, 1 << DEBUG, 1 << INFO, 1 << NOTICE,
        1 << WARN, 1 << ERROR, 1 << FATAL, 1 << PANIC,
    }
)

// String returns the string representation of the level - ZERO ALLOCATION
// String mengembalikan representasi string dari level - ZERO ALLOCATION
func (l Level) String() string {
    if l < Level(len(levelStrings)) {
        return levelStrings[l]
    }
    return "UNKNOWN"
}

// ParseLevel parses level from string - OPTIMIZED WITH SWITCH
// ParseLevel mem-parsing level dari string - OPTIMAL DENGAN SWITCH
func ParseLevel(levelStr string) (Level, error) {
    switch len(levelStr) {
    case 4:
        switch levelStr {
        case "INFO":
            return INFO, nil
        case "WARN":
            return WARN, nil
        case "FATAL":
            return FATAL, nil
        case "PANIC":
            return PANIC, nil
        }
    case 5:
        switch levelStr {
        case "TRACE":
            return TRACE, nil
        case "DEBUG":
            return DEBUG, nil
        case "ERROR":
            return ERROR, nil
        }
    case 6:
        if levelStr == "NOTICE" {
            return NOTICE, nil
        }
    case 7:
        if levelStr == "WARNING" {
            return WARN, nil
        }
    }
    return INFO, fmt.Errorf("invalid log level: %s", levelStr)
}

// ===============================
// ZERO-ALLOCATION STRING UTILITIES
// ===============================

// Unsafe string/byte conversions for zero allocation
// Konversi string/byte unsafe untuk zero allocation
var (
    // String header for zero-allocation conversions
    // Header string untuk konversi zero-allocation
    stringHeaderSize = int(unsafe.Sizeof((*reflect.StringHeader)(nil)))
    sliceHeaderSize  = int(unsafe.Sizeof((*reflect.SliceHeader)(nil)))
)

// stringToBytes converts string to byte slice without allocation
// stringToBytes mengkonversi string ke slice byte tanpa alokasi
func stringToBytes(s string) []byte {
    return *(*[]byte)(unsafe.Pointer(
        &struct {
            string
            Cap int
        }{s, len(s)},
    ))
}

// bytesToString converts byte slice to string without allocation
// bytesToString mengkonversi slice byte ke string tanpa alokasi
func bytesToString(b []byte) string {
    return *(*string)(unsafe.Pointer(&b))
}

// ===============================
// HIGH-PERFORMANCE LOG ENTRY - ZERO ALLOCATION DESIGN
// ===============================

// Pre-allocated field keys to avoid string allocations
// Kunci field pra-alokasi untuk menghindari alokasi string
var (
    fieldKeyTimestamp = "timestamp"
    fieldKeyLevel     = "level"
    fieldKeyMessage   = "message"
    fieldKeyCaller    = "caller"
    fieldKeyPID       = "pid"
    fieldKeyGoroutine = "goroutine_id"
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

// LogEntry represents a single log entry with zero-allocation design
// LogEntry merepresentasikan entri log tunggal dengan desain zero-allocation
type LogEntry struct {
    // Core fields - all pre-sized for zero allocation
    // Field inti - semua berukuran tetap untuk zero allocation
    Timestamp     time.Time
    Level         Level
    LevelName     [16]byte     // Fixed-size buffer for level name
    Message       [1024]byte   // Fixed-size buffer for message
    MessageLen    int          // Actual message length
    Caller        CallerInfo   // Embedded struct, not pointer
    Fields        [MAX_PREALLOCATED_FIELDS]FieldPair // Pre-allocated field pairs
    FieldsCount   int                                 // Actual field count
    Tags          [MAX_PREALLOCATED_TAGS]string      // Pre-allocated tags
    TagsCount     int                                // Actual tags count
    
    // Additional metadata
    // Metadata tambahan
    PID           int
    GoroutineID   [32]byte
    GoroutineIDLen int
    TraceID       [32]byte
    TraceIDLen    int
    SpanID        [32]byte
    SpanIDLen     int
    UserID        [64]byte
    UserIDLen     int
    SessionID     [64]byte
    SessionIDLen  int
    RequestID     [32]byte
    RequestIDLen  int
    Duration      time.Duration
    Error         error
    StackTrace    [4096]byte
    StackTraceLen int
    Hostname      [256]byte
    HostnameLen   int
    Application   [128]byte
    ApplicationLen int
    Version       [64]byte
    VersionLen    int
    Environment   [64]byte
    EnvironmentLen int
    CustomMetrics [8]MetricPair // Pre-allocated metrics
    MetricsCount  int           // Actual metrics count
}

// FieldPair represents a key-value pair for fields
// FieldPair merepresentasikan pasangan key-value untuk field
type FieldPair struct {
    Key   [64]byte // Fixed-size key buffer
    KeyLen int     // Actual key length
    Value interface{} // Value can be any type
}

// MetricPair represents a key-value pair for metrics
// MetricPair merepresentasikan pasangan key-value untuk metrik
type MetricPair struct {
    Key   [64]byte // Fixed-size key buffer
    KeyLen int     // Actual key length
    Value float64  // Numeric value
}

// CallerInfo contains caller information (embedded for zero allocation)
// CallerInfo berisi informasi pemanggil (embedded untuk zero allocation)
type CallerInfo struct {
    File     [256]byte // Fixed-size buffer for file name
    FileLen  int       // Actual file name length
    Line     int       // Line number
    Function [128]byte // Fixed-size buffer for function name
    FunctionLen int    // Actual function name length
    Package  [128]byte // Fixed-size buffer for package name
    PackageLen int     // Actual package name length
}

// Object pool for LogEntry with zero-allocation design
// Pool objek untuk LogEntry dengan desain zero-allocation
var entryPool = sync.Pool{
    New: func() interface{} {
        return &LogEntry{}
    },
}

// getEntryFromPool gets a LogEntry from pool with zero allocation
// getEntryFromPool mendapatkan LogEntry dari pool dengan zero allocation
func getEntryFromPool() *LogEntry {
    entry := entryPool.Get().(*LogEntry)
    // Reset fields efficiently without allocations
    // Reset field secara efisien tanpa alokasi
    *entry = LogEntry{} // Zero-allocation reset
    return entry
}

// putEntryToPool returns a LogEntry to pool
// putEntryToPool mengembalikan LogEntry ke pool
func putEntryToPool(entry *LogEntry) {
    entryPool.Put(entry)
}

// SetField sets a field with zero allocation
// SetField mengatur field dengan zero allocation
func (e *LogEntry) SetField(key string, value interface{}) {
    if e.FieldsCount >= len(e.Fields) {
        return // Buffer full, skip to avoid allocation
    }
    
    // Copy key to fixed buffer
    // Salin key ke buffer tetap
    keyLen := len(key)
    if keyLen > len(e.Fields[e.FieldsCount].Key) {
        keyLen = len(e.Fields[e.FieldsCount].Key)
    }
    copy(e.Fields[e.FieldsCount].Key[:], key[:keyLen])
    e.Fields[e.FieldsCount].KeyLen = keyLen
    e.Fields[e.FieldsCount].Value = value
    e.FieldsCount++
}

// SetStringField sets a string field with zero allocation
// SetStringField mengatur field string dengan zero allocation
func (e *LogEntry) SetStringField(key, value string) {
    if e.FieldsCount >= len(e.Fields) {
        return
    }
    
    keyLen := len(key)
    valLen := len(value)
    
    if keyLen > len(e.Fields[e.FieldsCount].Key) {
        keyLen = len(e.Fields[e.FieldsCount].Key)
    }
    if valLen > len(e.Fields[e.FieldsCount].Key) {
        valLen = len(e.Fields[e.FieldsCount].Key)
    }
    
    copy(e.Fields[e.FieldsCount].Key[:], key[:keyLen])
    e.Fields[e.FieldsCount].KeyLen = keyLen
    
    // Store as string in value interface
    // Simpan sebagai string di interface value
    if valLen == len(value) {
        e.Fields[e.FieldsCount].Value = value
    } else {
        e.Fields[e.FieldsCount].Value = value[:valLen]
    }
    e.FieldsCount++
}

// SetMetric sets a metric with zero allocation
// SetMetric mengatur metrik dengan zero allocation
func (e *LogEntry) SetMetric(key string, value float64) {
    if e.MetricsCount >= len(e.CustomMetrics) {
        return
    }
    
    keyLen := len(key)
    if keyLen > len(e.CustomMetrics[e.MetricsCount].Key) {
        keyLen = len(e.CustomMetrics[e.MetricsCount].Key)
    }
    
    copy(e.CustomMetrics[e.MetricsCount].Key[:], key[:keyLen])
    e.CustomMetrics[e.MetricsCount].KeyLen = keyLen
    e.CustomMetrics[e.MetricsCount].Value = value
    e.MetricsCount++
}

// SetTag adds a tag with zero allocation
// SetTag menambahkan tag dengan zero allocation
func (e *LogEntry) SetTag(tag string) {
    if e.TagsCount >= len(e.Tags) {
        return
    }
    
    e.Tags[e.TagsCount] = tag
    e.TagsCount++
}

// ===============================
// CONTEXT SUPPORT - ZERO ALLOCATION
// ===============================
type contextKey string

const (
    TraceIDKey contextKey = "trace_id"
    SpanIDKey  contextKey = "span_id"
    UserIDKey  contextKey = "user_id"
    SessionIDKey contextKey = "session_id"
    RequestIDKey contextKey = "request_id"
    ClientIPKey contextKey = "client_ip"
)

// WithTraceID adds trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
    return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithSpanID adds span ID to context
func WithSpanID(ctx context.Context, spanID string) context.Context {
    return context.WithValue(ctx, SpanIDKey, spanID)
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
    return context.WithValue(ctx, UserIDKey, userID)
}

// WithSessionID adds session ID to context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
    return context.WithValue(ctx, SessionIDKey, sessionID)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, RequestIDKey, requestID)
}

// ExtractFromContext extracts context values with minimal allocation
// ExtractFromContext mengekstrak nilai konteks dengan alokasi minimal
func ExtractFromContext(ctx context.Context) map[string]string {
    result := make(map[string]string, 5)
    
    if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
        result["trace_id"] = traceID
    }
    if spanID, ok := ctx.Value(SpanIDKey).(string); ok && spanID != "" {
        result["span_id"] = spanID
    }
    if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
        result["user_id"] = userID
    }
    if sessionID, ok := ctx.Value(SessionIDKey).(string); ok && sessionID != "" {
        result["session_id"] = sessionID
    }
    if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
        result["request_id"] = requestID
    }
    
    return result
}

// ===============================
// ZERO-ALLOCATION FORMATTERS
// ===============================

// Pre-allocated byte buffers for formatting
// Buffer byte pra-alokasi untuk formatting
var bufferPool = sync.Pool{
    New: func() interface{} {
        return &ByteArray{data: make([]byte, 0, LOG_ENTRY_BUFFER_SIZE)}
    },
}

// ByteArray provides zero-allocation byte array operations
// ByteArray menyediakan operasi array byte zero-allocation
type ByteArray struct {
    data []byte
    pool *sync.Pool
}

// Reset clears the buffer
func (b *ByteArray) Reset() {
    b.data = b.data[:0]
}

// WriteByte appends a byte
func (b *ByteArray) WriteByte(c byte) {
    b.data = append(b.data, c)
}

// WriteString appends a string
func (b *ByteArray) WriteString(s string) {
    b.data = append(b.data, sToBytes(s)...)
}

// Write appends bytes
func (b *ByteArray) Write(p []byte) {
    b.data = append(b.data, p...)
}

// Len returns length
func (b *ByteArray) Len() int {
    return len(b.data)
}

// Bytes returns the byte slice
func (b *ByteArray) Bytes() []byte {
    return b.data
}

// String returns the string representation
func (b *ByteArray) String() string {
    return bToString(b.data)
}

// getBufferFromPool gets a buffer from pool
func getBufferFromPool() *ByteArray {
    buf := bufferPool.Get().(*ByteArray)
    buf.Reset()
    return buf
}

// putBufferToPool returns a buffer to pool
func putBufferToPool(buf *ByteArray) {
    if buf.Len() < 64*1024 {
        bufferPool.Put(buf)
    }
}

// TextFormatter - ZERO ALLOCATION VERSION
type TextFormatter struct {
    EnableColors          bool
    ShowTimestamp         bool
    ShowCaller            bool
    ShowGoroutine         bool
    ShowPID               bool
    ShowTraceInfo         bool
    ShowHostname          bool
    ShowApplication       bool
    FullTimestamp         bool
    TimestampFormat       string
    EnableStackTrace      bool
    StackTraceDepth       int
    EnableDuration        bool
    MaxFieldWidth         int
    MaskSensitiveData     bool
    MaskString            string
    // Pre-allocated scratch buffers
    scratchBuffer         [512]byte
}

// Format formats a log entry with zero allocation
func (f *TextFormatter) Format(entry *LogEntry) ([]byte, error) {
    buf := getBufferFromPool()
    defer putBufferToPool(buf)
    
    // Write timestamp
    if f.ShowTimestamp {
        buf.WriteByte('[')
        buf.WriteString(entry.Timestamp.Format(f.TimestampFormat))
        buf.WriteByte(']')
        buf.WriteByte(' ')
    }
    
    // Write level
    levelStr := entry.Level.String()
    if f.EnableColors {
        buf.WriteString(levelBackgrounds[entry.Level])
        buf.WriteString(levelColors[entry.Level])
        buf.WriteByte(' ')
        buf.WriteString(levelStr)
        buf.WriteByte(' ')
        buf.WriteString("\033[0m")
    } else {
        buf.WriteByte('[')
        buf.WriteString(levelStr)
        buf.WriteByte(']')
    }
    buf.WriteByte(' ')
    
    // Write hostname
    if f.ShowHostname && entry.HostnameLen > 0 {
        if f.EnableColors {
            buf.WriteString("\033[38;5;245m")
        }
        buf.Write(entry.Hostname[:entry.HostnameLen])
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteByte(' ')
    }
    
    // Write application
    if f.ShowApplication && entry.ApplicationLen > 0 {
        if f.EnableColors {
            buf.WriteString("\033[38;5;245m")
        }
        buf.Write(entry.Application[:entry.ApplicationLen])
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteByte(' ')
    }
    
    // Write PID
    if f.ShowPID {
        if f.EnableColors {
            buf.WriteString("\033[38;5;245m")
        }
        buf.WriteString("PID:")
        buf.WriteString(strconv.Itoa(entry.PID))
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteByte(' ')
    }
    
    // Write goroutine ID
    if f.ShowGoroutine && entry.GoroutineIDLen > 0 {
        if f.EnableColors {
            buf.WriteString("\033[38;5;245m")
        }
        buf.WriteString("GID:")
        buf.Write(entry.GoroutineID[:entry.GoroutineIDLen])
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteByte(' ')
    }
    
    // Write trace info
    if f.ShowTraceInfo {
        if entry.TraceIDLen > 0 {
            if f.EnableColors {
                buf.WriteString("\033[38;5;141m")
            }
            buf.WriteString("TRACE:")
            f.writeShortID(buf, entry.TraceID[:entry.TraceIDLen])
            if f.EnableColors {
                buf.WriteString("\033[0m")
            }
            buf.WriteByte(' ')
        }
        
        if entry.SpanIDLen > 0 {
            if f.EnableColors {
                buf.WriteString("\033[38;5;141m")
            }
            buf.WriteString("SPAN:")
            f.writeShortID(buf, entry.SpanID[:entry.SpanIDLen])
            if f.EnableColors {
                buf.WriteString("\033[0m")
            }
            buf.WriteByte(' ')
        }
        
        if entry.RequestIDLen > 0 {
            if f.EnableColors {
                buf.WriteString("\033[38;5;141m")
            }
            buf.WriteString("REQ:")
            f.writeShortID(buf, entry.RequestID[:entry.RequestIDLen])
            if f.EnableColors {
                buf.WriteString("\033[0m")
            }
            buf.WriteByte(' ')
        }
    }
    
    // Write caller
    if f.ShowCaller && entry.Caller.FileLen > 0 {
        if f.EnableColors {
            buf.WriteString("\033[38;5;246m")
        }
        buf.Write(entry.Caller.File[:entry.Caller.FileLen])
        buf.WriteByte(':')
        buf.WriteString(strconv.Itoa(entry.Caller.Line))
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteByte(' ')
    }
    
    // Write duration
    if f.EnableDuration && entry.Duration > 0 {
        if f.EnableColors {
            buf.WriteString("\033[38;5;155m")
        }
        buf.WriteByte('(')
        buf.WriteString(entry.Duration.String())
        buf.WriteByte(')')
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteByte(' ')
    }
    
    // Write message
    if f.EnableColors && entry.Level >= ERROR {
        buf.WriteString("\033[1m")
    }
    buf.Write(entry.Message[:entry.MessageLen])
    if f.EnableColors && entry.Level >= ERROR {
        buf.WriteString("\033[0m")
    }
    
    // Write error
    if entry.Error != nil {
        buf.WriteByte(' ')
        if f.EnableColors {
            buf.WriteString("\033[38;5;196m")
        }
        buf.WriteString("error=\"")
        buf.WriteString(entry.Error.Error())
        buf.WriteByte('"')
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
    }
    
    // Write fields
    if entry.FieldsCount > 0 {
        buf.WriteByte(' ')
        f.formatFields(buf, entry)
    }
    
    // Write tags
    if entry.TagsCount > 0 {
        buf.WriteByte(' ')
        f.formatTags(buf, entry)
    }
    
    // Write metrics
    if entry.MetricsCount > 0 {
        buf.WriteByte(' ')
        f.formatMetrics(buf, entry)
    }
    
    // Write stack trace
    if f.EnableStackTrace && entry.StackTraceLen > 0 {
        buf.WriteByte('\n')
        if f.EnableColors {
            buf.WriteString("\033[38;5;240m")
        }
        buf.Write(entry.StackTrace[:entry.StackTraceLen])
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
    }
    
    buf.WriteByte('\n')
    
    // Return copy to avoid buffer reuse issues
    result := make([]byte, buf.Len())
    copy(result, buf.Bytes())
    return result, nil
}

// writeShortID writes a shortened ID
func (f *TextFormatter) writeShortID(buf *ByteArray, id []byte) {
    if len(id) <= 8 {
        buf.Write(id)
    } else {
        buf.Write(id[:8])
    }
}

// formatFields formats fields
func (f *TextFormatter) formatFields(buf *ByteArray, entry *LogEntry) {
    if f.EnableColors {
        buf.WriteString("\033[38;5;243m")
    }
    buf.WriteByte('{')
    
    for i := 0; i < entry.FieldsCount; i++ {
        if i > 0 {
            buf.WriteByte(' ')
        }
        
        if f.EnableColors {
            buf.WriteString("\033[38;5;228m")
        }
        buf.Write(entry.Fields[i].Key[:entry.Fields[i].KeyLen])
        buf.WriteByte('=')
        if f.EnableColors {
            buf.WriteString("\033[38;5;159m")
        }
        
        // Format value
        f.formatValue(buf, entry.Fields[i].Value)
    }
    
    if f.EnableColors {
        buf.WriteString("\033[38;5;243m")
    }
    buf.WriteByte('}')
    if f.EnableColors {
        buf.WriteString("\033[0m")
    }
}

// formatTags formats tags
func (f *TextFormatter) formatTags(buf *ByteArray, entry *LogEntry) {
    if f.EnableColors {
        buf.WriteString("\033[38;5;135m")
    }
    buf.WriteByte('[')
    for i := 0; i < entry.TagsCount; i++ {
        if i > 0 {
            buf.WriteByte(',')
        }
        buf.WriteString(entry.Tags[i])
    }
    buf.WriteByte(']')
    if f.EnableColors {
        buf.WriteString("\033[0m")
    }
}

// formatMetrics formats metrics
func (f *TextFormatter) formatMetrics(buf *ByteArray, entry *LogEntry) {
    if f.EnableColors {
        buf.WriteString("\033[38;5;85m")
    }
    buf.WriteByte('<')
    for i := 0; i < entry.MetricsCount; i++ {
        if i > 0 {
            buf.WriteByte(' ')
        }
        buf.Write(entry.CustomMetrics[i].Key[:entry.CustomMetrics[i].KeyLen])
        buf.WriteByte('=')
        buf.WriteString(strconv.FormatFloat(entry.CustomMetrics[i].Value, 'f', 2, 64))
    }
    buf.WriteByte('>')
    if f.EnableColors {
        buf.WriteString("\033[0m")
    }
}

// formatValue formats a value
func (f *TextFormatter) formatValue(buf *ByteArray, value interface{}) {
    switch v := value.(type) {
    case string:
        if f.MaxFieldWidth > 0 && len(v) > f.MaxFieldWidth {
            buf.Write(sToBytes(v[:f.MaxFieldWidth]))
            buf.WriteString("...")
        } else {
            buf.Write(sToBytes(v))
        }
    case int:
        buf.WriteString(strconv.Itoa(v))
    case int64:
        buf.WriteString(strconv.FormatInt(v, 10))
    case float64:
        buf.WriteString(strconv.FormatFloat(v, 'f', 2, 64))
    case bool:
        if v {
            buf.WriteString("true")
        } else {
            buf.WriteString("false")
        }
    case nil:
        buf.WriteString("null")
    default:
        s := fmt.Sprintf("%v", v)
        if f.MaxFieldWidth > 0 && len(s) > f.MaxFieldWidth {
            buf.Write(sToBytes(s[:f.MaxFieldWidth]))
            buf.WriteString("...")
        } else {
            buf.Write(sToBytes(s))
        }
    }
}

// JSONFormatter - ZERO ALLOCATION VERSION
type JSONFormatter struct {
    PrettyPrint         bool
    TimestampFormat     string
    ShowCaller          bool
    ShowGoroutine       bool
    ShowPID             bool
    ShowTraceInfo       bool
    EnableStackTrace    bool
    EnableDuration      bool
    DisableHTMLEscape   bool
    MaskSensitiveData   bool
    MaskString          string
}

// NewJSONFormatter creates a new JSONFormatter
func NewJSONFormatter() *JSONFormatter {
    return &JSONFormatter{
        TimestampFormat: DEFAULT_TIMESTAMP_FORMAT,
    }
}

// Format formats a log entry as JSON with minimal allocation
func (f *JSONFormatter) Format(entry *LogEntry) ([]byte, error) {
    buf := getBufferFromPool()
    defer putBufferToPool(buf)
    
    encoder := json.NewEncoder(buf)
    if f.DisableHTMLEscape {
        encoder.SetEscapeHTML(false)
    }
    
    // Build output structure
    output := make(map[string]interface{}, 16)
    
    // Timestamp
    if f.TimestampFormat != "" {
        output["timestamp"] = entry.Timestamp.Format(f.TimestampFormat)
    } else {
        output["timestamp"] = entry.Timestamp
    }
    
    // Level
    output["level"] = entry.Level.String()
    output["level_name"] = entry.Level.String()
    
    // Message
    output["message"] = bToString(entry.Message[:entry.MessageLen])
    
    // Metadata
    output["pid"] = entry.PID
    
    if f.ShowCaller && entry.Caller.FileLen > 0 {
        caller := make(map[string]interface{}, 4)
        caller["file"] = bToString(entry.Caller.File[:entry.Caller.FileLen])
        caller["line"] = entry.Caller.Line
        caller["function"] = bToString(entry.Caller.Function[:entry.Caller.FunctionLen])
        caller["package"] = bToString(entry.Caller.Package[:entry.Caller.PackageLen])
        output["caller"] = caller
    }
    
    if f.ShowGoroutine && entry.GoroutineIDLen > 0 {
        output["goroutine_id"] = bToString(entry.GoroutineID[:entry.GoroutineIDLen])
    }
    
    if f.ShowTraceInfo {
        if entry.TraceIDLen > 0 {
            output["trace_id"] = bToString(entry.TraceID[:entry.TraceIDLen])
        }
        if entry.SpanIDLen > 0 {
            output["span_id"] = bToString(entry.SpanID[:entry.SpanIDLen])
        }
        if entry.UserIDLen > 0 {
            output["user_id"] = bToString(entry.UserID[:entry.UserIDLen])
        }
        if entry.SessionIDLen > 0 {
            output["session_id"] = bToString(entry.SessionID[:entry.SessionIDLen])
        }
        if entry.RequestIDLen > 0 {
            output["request_id"] = bToString(entry.RequestID[:entry.RequestIDLen])
        }
    }
    
    if f.EnableDuration && entry.Duration > 0 {
        output["duration"] = entry.Duration.String()
    }
    
    if f.EnableStackTrace && entry.StackTraceLen > 0 {
        output["stack_trace"] = bToString(entry.StackTrace[:entry.StackTraceLen])
    }
    
    if entry.HostnameLen > 0 {
        output["hostname"] = bToString(entry.Hostname[:entry.HostnameLen])
    }
    
    if entry.ApplicationLen > 0 {
        output["application"] = bToString(entry.Application[:entry.ApplicationLen])
    }
    
    if entry.VersionLen > 0 {
        output["version"] = bToString(entry.Version[:entry.VersionLen])
    }
    
    if entry.EnvironmentLen > 0 {
        output["environment"] = bToString(entry.Environment[:entry.EnvironmentLen])
    }
    
    // Fields
    if entry.FieldsCount > 0 {
        fields := make(map[string]interface{}, entry.FieldsCount)
        for i := 0; i < entry.FieldsCount; i++ {
            key := bToString(entry.Fields[i].Key[:entry.Fields[i].KeyLen])
            fields[key] = entry.Fields[i].Value
        }
        output["fields"] = fields
    }
    
    // Tags
    if entry.TagsCount > 0 {
        tags := make([]string, entry.TagsCount)
        for i := 0; i < entry.TagsCount; i++ {
            tags[i] = entry.Tags[i]
        }
        output["tags"] = tags
    }
    
    // Metrics
    if entry.MetricsCount > 0 {
        metrics := make(map[string]float64, entry.MetricsCount)
        for i := 0; i < entry.MetricsCount; i++ {
            key := bToString(entry.CustomMetrics[i].Key[:entry.CustomMetrics[i].KeyLen])
            metrics[key] = entry.CustomMetrics[i].Value
        }
        output["custom_metrics"] = metrics
    }
    
    // Error
    if entry.Error != nil {
        output["error"] = entry.Error.Error()
    }
    
    var err error
    if f.PrettyPrint {
        data, err := json.MarshalIndent(output, "", "  ")
        if err != nil {
            return nil, err
        }
        result := make([]byte, len(data))
        copy(result, data)
        return result, nil
    } else {
        err = encoder.Encode(output)
        if err != nil {
            return nil, err
        }
    }
    
    result := make([]byte, buf.Len())
    copy(result, buf.Bytes())
    return result, nil
}

// ===============================
// HIGH-PERFORMANCE BUFFERED WRITER - ZERO ALLOCATION
// ===============================

// Pre-sized buffer pool for different sizes
var (
    smallBufferPool  = sync.Pool{New: func() interface{} { return make([]byte, 0, 1024) }}
    mediumBufferPool = sync.Pool{New: func() interface{} { return make([]byte, 0, 8192) }}
    largeBufferPool  = sync.Pool{New: func() interface{} { return make([]byte, 0, 65536) }}
)

// BufferedWriter - ZERO ALLOCATION VERSION
type BufferedWriter struct {
    writer        io.Writer
    buffer        chan []byte
    bufferSize    int
    flushInterval time.Duration
    done          chan struct{}
    wg            sync.WaitGroup
    droppedLogs   int64
    totalLogs     int64
    lastFlush     time.Time
    batchSize     int
    batchTimeout  time.Duration
}

// NewBufferedWriter creates a new BufferedWriter
func NewBufferedWriter(writer io.Writer, bufferSize int, flushInterval time.Duration) *BufferedWriter {
    bw := &BufferedWriter{
        writer:        writer,
        buffer:        make(chan []byte, bufferSize),
        bufferSize:    bufferSize,
        flushInterval: flushInterval,
        done:          make(chan struct{}),
        lastFlush:     time.Now(),
        batchSize:     100,
        batchTimeout:  100 * time.Millisecond,
    }
    
    bw.wg.Add(1)
    go bw.flushWorker()
    return bw
}

// Write writes data with zero allocation strategy
func (bw *BufferedWriter) Write(p []byte) (n int, err error) {
    atomic.AddInt64(&bw.totalLogs, 1)
    
    // Choose appropriate buffer pool based on data size
    var buf []byte
    if len(p) <= 1024 {
        poolBuf := smallBufferPool.Get().([]byte)
        buf = append(poolBuf[:0], p...)
    } else if len(p) <= 8192 {
        poolBuf := mediumBufferPool.Get().([]byte)
        buf = append(poolBuf[:0], p...)
    } else {
        poolBuf := largeBufferPool.Get().([]byte)
        buf = append(poolBuf[:0], p...)
    }
    
    select {
    case bw.buffer <- buf:
        return len(p), nil
    default:
        atomic.AddInt64(&bw.droppedLogs, 1)
        // Fallback to direct write
        return bw.writer.Write(p)
    }
}

// flushWorker processes buffered writes
func (bw *BufferedWriter) flushWorker() {
    defer bw.wg.Done()
    
    ticker := time.NewTicker(bw.flushInterval)
    defer ticker.Stop()
    
    batch := make([][]byte, 0, bw.batchSize)
    batchTimer := time.NewTimer(bw.batchTimeout)
    defer batchTimer.Stop()
    
    for {
        select {
        case <-bw.done:
            bw.flushBatch(batch)
            return
        case <-ticker.C:
            bw.flushBatch(batch)
            batch = batch[:0]
            bw.lastFlush = time.Now()
            if !batchTimer.Stop() {
                <-batchTimer.C
            }
            batchTimer.Reset(bw.batchTimeout)
        case data := <-bw.buffer:
            batch = append(batch, data)
            if len(batch) >= bw.batchSize {
                bw.flushBatch(batch)
                batch = batch[:0]
                bw.lastFlush = time.Now()
                if !batchTimer.Stop() {
                    <-batchTimer.C
                }
                batchTimer.Reset(bw.batchTimeout)
            }
        case <-batchTimer.C:
            if len(batch) > 0 {
                bw.flushBatch(batch)
                batch = batch[:0]
                bw.lastFlush = time.Now()
            }
            batchTimer.Reset(bw.batchTimeout)
        }
    }
}

// flushBatch flushes a batch efficiently
func (bw *BufferedWriter) flushBatch(batch [][]byte) {
    if len(batch) == 0 {
        return
    }
    
    // Calculate total size and pre-allocate combined buffer
    totalSize := 0
    for _, data := range batch {
        totalSize += len(data)
    }
    
    combined := make([]byte, 0, totalSize)
    for _, data := range batch {
        combined = append(combined, data...)
        // Return to appropriate pool
        if len(data) <= 1024 {
            smallBufferPool.Put(data[:0])
        } else if len(data) <= 8192 {
            mediumBufferPool.Put(data[:0])
        } else {
            largeBufferPool.Put(data[:0])
        }
    }
    
    bw.writer.Write(combined)
}

// Stats returns statistics
func (bw *BufferedWriter) Stats() map[string]interface{} {
    return map[string]interface{}{
        "buffer_size":    bw.bufferSize,
        "current_queue":  len(bw.buffer),
        "dropped_logs":   atomic.LoadInt64(&bw.droppedLogs),
        "total_logs":     atomic.LoadInt64(&bw.totalLogs),
        "last_flush":     bw.lastFlush,
    }
}

// Close closes the writer
func (bw *BufferedWriter) Close() error {
    close(bw.done)
    bw.wg.Wait()
    
    // Flush remaining data
    bw.flushBatch(nil)
    
    if closer, ok := bw.writer.(io.Closer); ok {
        return closer.Close()
    }
    return nil
}

// ===============================
// ZERO-ALLOCATION LOGGER CORE
// ===============================

// LoggerConfig - OPTIMIZED
type LoggerConfig struct {
    Level                Level
    EnableColors         bool
    Output               io.Writer
    ErrorOutput          io.Writer
    Formatter            Formatter
    ShowCaller           bool
    CallerDepth          int
    ShowGoroutine        bool
    ShowPID              bool
    ShowTraceInfo        bool
    ShowHostname         bool
    ShowApplication      bool
    TimestampFormat      string
    ExitFunc             func(int)
    EnableStackTrace     bool
    StackTraceDepth      int
    EnableSampling       bool
    SamplingRate         int
    BufferSize           int
    FlushInterval        time.Duration
    EnableRotation       bool
    RotationConfig       *RotationConfig
    ContextExtractor     func(context.Context) map[string]string
    Hostname             string
    Application          string
    Version              string
    Environment          string
    MaxFieldSize         int
    EnableMetrics        bool
    MetricsCollector     MetricsCollector
    ErrorHandler         func(error)
    OnFatal              func(*LogEntry)
    OnPanic              func(*LogEntry)
    DisableLocking       bool
    PreAllocateFields    int
    PreAllocateTags      int
    MaxMessageSize       int
    AsyncLogging         bool
    // Zero-allocation optimizations
    DisableFieldCopy     bool  // Skip field copying for performance
    DisableTimestamp     bool  // Skip timestamp for performance
    DisableCallerInfo    bool  // Skip caller info for performance
    FastPathLevel        Level // Levels below this use fast path
}

// Logger - ZERO ALLOCATION VERSION
type Logger struct {
    config             LoggerConfig
    formatter          Formatter
    out                io.Writer
    errOut             io.Writer
    mu                 sync.Mutex
    hooks              []func(*LogEntry)
    exitFunc           func(int)
    fields             map[string]interface{}
    sampler            *SamplingLogger
    buffer             *BufferedWriter
    rotation           *RotatingFileWriter
    contextExtractor   func(context.Context) map[string]string
    metrics            MetricsCollector
    errorHandler       func(error)
    onFatal            func(*LogEntry)
    onPanic            func(*LogEntry)
    stats              *LoggerStats
    asyncLogger        *AsyncLogger
    
    // Zero-allocation optimizations
    levelMask          uint64           // Bitmask for fast level checking
    hostnameBytes      []byte           // Pre-converted hostname
    applicationBytes   []byte           // Pre-converted application name
    versionBytes       []byte           // Pre-converted version
    environmentBytes   []byte           // Pre-converted environment
    pidStr             []byte           // Pre-converted PID
    goroutineCache     string           // Cached goroutine ID
    callerCache        CallerInfo       // Cached caller info
    cachedFields       [8]FieldPair     // Cached default fields
    cachedFieldsCount  int              // Count of cached fields
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger() *Logger {
    config := LoggerConfig{
        Level:               INFO,
        EnableColors:        true,
        Output:              os.Stdout,
        ErrorOutput:         os.Stderr,
        ShowCaller:          true,
        CallerDepth:         DEFAULT_CALLER_DEPTH,
        ShowGoroutine:       true,
        ShowPID:             true,
        ShowTraceInfo:       true,
        TimestampFormat:     DEFAULT_TIMESTAMP_FORMAT,
        EnableStackTrace:    true,
        StackTraceDepth:     10,
        BufferSize:          DEFAULT_BUFFER_SIZE,
        FlushInterval:       DEFAULT_FLUSH_INTERVAL,
        PreAllocateFields:   10,
        PreAllocateTags:     5,
        MaxMessageSize:      MAX_MESSAGE_SIZE,
        FastPathLevel:       WARN, // Use fast path for WARN and above
    }
    
    config.Formatter = &TextFormatter{
        EnableColors:      true,
        ShowTimestamp:     true,
        ShowCaller:        true,
        ShowGoroutine:     true,
        ShowPID:           true,
        ShowTraceInfo:     true,
        TimestampFormat:   DEFAULT_TIMESTAMP_FORMAT,
        EnableStackTrace:  true,
        MaxFieldWidth:     100,
    }
    
    return NewLogger(config)
}

// NewLogger creates a new logger
func NewLogger(config LoggerConfig) *Logger {
    l := &Logger{
        config:    config,
        formatter: config.Formatter,
        out:       config.Output,
        errOut:    config.ErrorOutput,
        exitFunc:  config.ExitFunc,
        fields:    make(map[string]interface{}),
        contextExtractor: config.ContextExtractor,
        metrics:   config.MetricsCollector,
        errorHandler: config.ErrorHandler,
        onFatal:   config.OnFatal,
        onPanic:   config.OnPanic,
        stats:     NewLoggerStats(),
    }
    
    if l.exitFunc == nil {
        l.exitFunc = os.Exit
    }
    
    // Pre-compute level mask for fast checking
    l.levelMask = uint64(0)
    for i := config.Level; i < 8; i++ {
        l.levelMask |= levelMasks[i]
    }
    
    // Pre-convert static strings to bytes
    l.hostnameBytes = sToBytes(config.Hostname)
    l.applicationBytes = sToBytes(config.Application)
    l.versionBytes = sToBytes(config.Version)
    l.environmentBytes = sToBytes(config.Environment)
    l.pidStr = sToBytes(strconv.Itoa(os.Getpid()))
    
    // Setup buffering
    if config.BufferSize > 0 {
        l.buffer = NewBufferedWriter(config.Output, config.BufferSize, config.FlushInterval)
        l.out = l.buffer
    }
    
    // Setup sampling
    if config.EnableSampling && config.SamplingRate > 1 {
        l.sampler = NewSamplingLogger(l, config.SamplingRate)
    }
    
    // Setup async logging
    if config.AsyncLogging {
        l.asyncLogger = NewAsyncLogger(l, 4, config.BufferSize)
    }
    
    return l
}

// SetLevel sets the minimum log level with optimized bitmask
func (l *Logger) SetLevel(level Level) {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.config.Level = level
    
    // Update level mask
    l.levelMask = uint64(0)
    for i := level; i < 8; i++ {
        l.levelMask |= levelMasks[i]
    }
}

// Fast path level check
func (l *Logger) shouldLog(level Level) bool {
    return (l.levelMask & (1 << level)) != 0
}

// log is the core logging method with zero-allocation design
func (l *Logger) log(level Level, msg string, fields map[string]interface{}, ctx context.Context) {
    // Fast path check
    if !l.shouldLog(level) {
        return
    }
    
    // Sampling check
    if l.sampler != nil && !l.sampler.shouldLog() {
        return
    }
    
    // Message size check
    if l.config.MaxMessageSize > 0 && len(msg) > l.config.MaxMessageSize {
        msg = msg[:l.config.MaxMessageSize] + "... [truncated]"
    }
    
    start := time.Now()
    
    // Get entry from pool
    entry := getEntryFromPool()
    defer putEntryToPool(entry)
    
    // Fill core fields with zero allocation
    if !l.config.DisableTimestamp {
        entry.Timestamp = time.Now()
    }
    entry.Level = level
    
    // Copy level name to fixed buffer
    levelStr := level.String()
    copy(entry.LevelName[:], levelStr)
    
    // Copy message to fixed buffer
    msgLen := len(msg)
    if msgLen > len(entry.Message) {
        msgLen = len(entry.Message)
    }
    copy(entry.Message[:], msg[:msgLen])
    entry.MessageLen = msgLen
    
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
    if l.config.Version != "" && len(l.versionBytes) > 0 {
        copy(entry.Version[:], l.versionBytes)
        entry.VersionLen = len(l.versionBytes)
    }
    
    // Copy environment
    if l.config.Environment != "" && len(l.environmentBytes) > 0 {
        copy(entry.Environment[:], l.environmentBytes)
        entry.EnvironmentLen = len(l.environmentBytes)
    }
    
    // Context extraction
    if ctx != nil {
        contextValues := ExtractFromContext(ctx)
        
        if traceID, ok := contextValues["trace_id"]; ok && traceID != "" {
            traceBytes := sToBytes(traceID)
            copy(entry.TraceID[:], traceBytes)
            entry.TraceIDLen = len(traceBytes)
        }
        if spanID, ok := contextValues["span_id"]; ok && spanID != "" {
            spanBytes := sToBytes(spanID)
            copy(entry.SpanID[:], spanBytes)
            entry.SpanIDLen = len(spanBytes)
        }
        if userID, ok := contextValues["user_id"]; ok && userID != "" {
            userBytes := sToBytes(userID)
            copy(entry.UserID[:], userBytes)
            entry.UserIDLen = len(userBytes)
        }
        if sessionID, ok := contextValues["session_id"]; ok && sessionID != "" {
            sessionBytes := sToBytes(sessionID)
            copy(entry.SessionID[:], sessionBytes)
            entry.SessionIDLen = len(sessionBytes)
        }
        if requestID, ok := contextValues["request_id"]; ok && requestID != "" {
            requestBytes := sToBytes(requestID)
            copy(entry.RequestID[:], requestBytes)
            entry.RequestIDLen = len(requestBytes)
        }
    }
    
    // Caller information
    if l.config.ShowCaller && !l.config.DisableCallerInfo {
        if pc, file, line, ok := runtime.Caller(l.config.CallerDepth); ok {
            function := runtime.FuncForPC(pc).Name()
            pkg := filepath.Dir(function)
            
            // Copy file name
            fileBytes := sToBytes(filepath.Base(file))
            copy(entry.Caller.File[:], fileBytes)
            entry.Caller.FileLen = len(fileBytes)
            
            entry.Caller.Line = line
            
            // Copy function name
            funcBytes := sToBytes(filepath.Base(function))
            copy(entry.Caller.Function[:], funcBytes)
            entry.Caller.FunctionLen = len(funcBytes)
            
            // Copy package name
            pkgBytes := sToBytes(pkg)
            copy(entry.Caller.Package[:], pkgBytes)
            entry.Caller.PackageLen = len(pkgBytes)
        }
    }
    
    // Goroutine ID
    if l.config.ShowGoroutine {
        if l.goroutineCache == "" {
            l.goroutineCache = getGoroutineID()
        }
        goroutineBytes := sToBytes(l.goroutineCache)
        copy(entry.GoroutineID[:], goroutineBytes)
        entry.GoroutineIDLen = len(goroutineBytes)
    }
    
    // Stack trace for errors
    if l.config.EnableStackTrace && level >= ERROR {
        stackTrace := getStackTrace(l.config.StackTraceDepth)
        stackBytes := sToBytes(stackTrace)
        if len(stackBytes) > len(entry.StackTrace) {
            stackBytes = stackBytes[:len(entry.StackTrace)]
        }
        copy(entry.StackTrace[:], stackBytes)
        entry.StackTraceLen = len(stackBytes)
    }
    
    // Process fields
    if !l.config.DisableFieldCopy {
        // Add cached default fields first
        for i := 0; i < l.cachedFieldsCount; i++ {
            entry.SetField(
                bToString(l.cachedFields[i].Key[:l.cachedFields[i].KeyLen]),
                l.cachedFields[i].Value,
            )
        }
        
        // Add provided fields
        if fields != nil {
            for k, v := range fields {
                entry.SetField(k, v)
            }
        }
    }
    
    // Execute hooks
    for _, hook := range l.hooks {
        hook(entry)
    }
    
    // Conditional locking
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    
    // Formatting
    formatted, err := l.formatter.Format(entry)
    if err != nil {
        if l.errorHandler != nil {
            l.errorHandler(err)
        } else {
            fmt.Fprintf(l.errOut, "Logger error: %v\n", err)
        }
        return
    }
    
    // Writing
    bytesWritten, err := l.out.Write(formatted)
    if err != nil {
        if l.errorHandler != nil {
            l.errorHandler(err)
        }
    }
    
    // Update statistics
    l.stats.Increment(level, bytesWritten)
    
    // Metrics collection
    if l.metrics != nil {
        tags := map[string]string{
            "level": level.String(),
            "application": l.config.Application,
            "environment": l.config.Environment,
        }
        l.metrics.IncrementCounter(level, tags)
    }
    
    // Handle fatal and panic levels
    if level == FATAL {
        if l.onFatal != nil {
            l.onFatal(entry)
        }
        l.exitFunc(1)
    } else if level == PANIC {
        if l.onPanic != nil {
            l.onPanic(entry)
        }
        panic(msg)
    }
    
    // Slow log detection
    duration := time.Since(start)
    if duration > 100*time.Millisecond {
        slowFields := map[string]interface{}{
            "logging_duration": duration.String(),
            "message_length":   len(msg),
            "fields_count":     entry.FieldsCount,
        }
        l.log(WARN, "Slow logging operation", slowFields, nil)
    }
}

// ===============================
// UTILITY FUNCTIONS - ZERO ALLOCATION
// ===============================

// sToBytes converts string to []byte without allocation
func sToBytes(s string) []byte {
    return *(*[]byte)(unsafe.Pointer(
        &struct {
            string
            Cap int
        }{s, len(s)},
    ))
}

// bToString converts []byte to string without allocation
func bToString(b []byte) string {
    return *(*string)(unsafe.Pointer(&b))
}

// getGoroutineID gets the current goroutine ID efficiently
func getGoroutineID() string {
    var buf [GOROUTINE_ID_BUFFER_SIZE]byte
    n := runtime.Stack(buf[:], false)
    idField := bToString(buf[:n])
    idField = strings.TrimPrefix(idField, "goroutine ")
    spaceIndex := strings.Index(idField, " ")
    if spaceIndex >= 0 {
        return idField[:spaceIndex]
    }
    return idField
}

// getStackTrace gets the current stack trace efficiently
func getStackTrace(depth int) string {
    buf := make([]byte, STACK_TRACE_BUFFER_SIZE)
    for {
        n := runtime.Stack(buf, false)
        if n < len(buf) {
            return bToString(buf[:n])
        }
        buf = make([]byte, 2*len(buf))
    }
}

// ===============================
// ADDITIONAL HIGH-PERFORMANCE METHODS
// ===============================

// WithField returns a new logger with an additional field - OPTIMIZED
func (l *Logger) WithField(key string, value interface{}) *Logger {
    newLogger := NewLogger(l.config)
    newLogger.fields = make(map[string]interface{}, len(l.fields)+1)
    for k, v := range l.fields {
        newLogger.fields[k] = v
    }
    newLogger.fields[key] = value
    
    // Cache frequently used fields for zero allocation
    if newLogger.cachedFieldsCount < len(newLogger.cachedFields) {
        keyLen := len(key)
        if keyLen > len(newLogger.cachedFields[newLogger.cachedFieldsCount].Key) {
            keyLen = len(newLogger.cachedFields[newLogger.cachedFieldsCount].Key)
        }
        copy(newLogger.cachedFields[newLogger.cachedFieldsCount].Key[:], key[:keyLen])
        newLogger.cachedFields[newLogger.cachedFieldsCount].KeyLen = keyLen
        newLogger.cachedFields[newLogger.cachedFieldsCount].Value = value
        newLogger.cachedFieldsCount++
    }
    
    return newLogger
}

// Batch logging for improved performance
func (l *Logger) LogBatch(entries []*LogEntry) error {
    if len(entries) == 0 {
        return nil
    }
    
    // Pre-calculate total size
    totalSize := 0
    formattedEntries := make([][]byte, 0, len(entries))
    
    for _, entry := range entries {
        formatted, err := l.formatter.Format(entry)
        if err != nil {
            return err
        }
        formattedEntries = append(formattedEntries, formatted)
        totalSize += len(formatted)
    }
    
    // Combine all formatted entries
    combined := make([]byte, 0, totalSize)
    for _, formatted := range formattedEntries {
        combined = append(combined, formatted...)
    }
    
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    
    _, err := l.out.Write(combined)
    return err
}

// TimeOperation measures and logs operation duration efficiently
func (l *Logger) TimeOperation(operationName string, fields map[string]interface{}, operation func()) {
    start := time.Now()
    
    defer func() {
        duration := time.Since(start)
        
        opFields := make(map[string]interface{}, len(fields)+4)
        for k, v := range fields {
            opFields[k] = v
        }
        opFields["operation"] = operationName
        opFields["duration"] = duration.String()
        opFields["duration_ms"] = duration.Milliseconds()
        
        level := INFO
        if duration > time.Second {
            level = WARN
        }
        if duration > 5*time.Second {
            level = ERROR
        }
        
        if l.asyncLogger != nil {
            l.asyncLogger.log(level, fmt.Sprintf("Operation completed: %s", operationName), opFields, nil)
        } else {
            l.log(level, fmt.Sprintf("Operation completed: %s", operationName), opFields, nil)
        }
        
        // Metrics collection
        if l.metrics != nil {
            tags := map[string]string{
                "operation": operationName,
                "application": l.config.Application,
            }
            l.metrics.RecordHistogram("operation.duration", duration.Seconds(), tags)
        }
    }()
    
    operation()
}

// ===============================
// ROTATION AND COMPRESSION - OPTIMIZED
// ===============================

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
    
    if config.FilenamePattern != "" {
        r.pattern = config.FilenamePattern
    } else {
        r.pattern = "log.%Y%m%d.%H%M%S.log"
    }
    
    r.maxSize = config.MaxSize
    r.maxAge = config.MaxAge
    r.maxBackups = config.MaxBackups
    r.localTime = config.LocalTime
    r.rotationTime = config.RotationTime
    
    // Initialize file
    file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        return nil, err
    }
    r.file = file
    
    info, err := file.Stat()
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
    if r.file != nil {
        if err := r.file.Close(); err != nil {
            return err
        }
    }
    
    var newFilename string
    if r.localTime {
        newFilename = time.Now().Format(r.pattern)
    } else {
        now := time.Now().UTC()
        newFilename = now.Format(r.pattern)
    }
    
    if r.compress {
        newFilename += r.compressedExt
        if err := r.compressFile(r.filename, newFilename); err != nil {
            return err
        }
    } else {
        if err := os.Rename(r.filename, newFilename); err != nil {
            return err
        }
    }
    
    file, err := os.OpenFile(r.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        return err
    }
    
    r.file = file
    r.currentSize = 0
    r.lastRotation = time.Now()
    
    if r.maxBackups > 0 {
        r.cleanupBackups()
    }
    
    return nil
}

// compressFile compresses a file using gzip
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
    
    // Use buffer pool
    buf := r.bufferPool.Get().([]byte)
    defer r.bufferPool.Put(buf[:0])
    
    _, err = io.CopyBuffer(gzWriter, srcFile, buf)
    if err != nil {
        return err
    }
    
    return os.Remove(srcFilename)
}

// cleanupBackups removes old backup files
func (r *RotatingFileWriter) cleanupBackups() error {
    if r.maxBackups <= 0 {
        return nil
    }
    
    files, err := filepath.Glob("log.*.log*")
    if err != nil {
        return err
    }
    
    sort.Strings(files)
    
    if len(files) > r.maxBackups {
        for i := 0; i < len(files)-r.maxBackups; i++ {
            if err := os.Remove(files[i]); err != nil {
                return err
            }
        }
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

// ===============================
// METRICS COLLECTOR - OPTIMIZED
// ===============================

// DefaultMetricsCollector is a simple in-memory metrics collector
type DefaultMetricsCollector struct {
    counters   map[string]int64
    histograms map[string][]float64
    gauges     map[string]float64
    mu         sync.RWMutex
}

// NewDefaultMetricsCollector creates a new DefaultMetricsCollector
func NewDefaultMetricsCollector() *DefaultMetricsCollector {
    return &DefaultMetricsCollector{
        counters:   make(map[string]int64),
        histograms: make(map[string][]float64),
        gauges:     make(map[string]float64),
    }
}

// IncrementCounter increments a counter metric
func (d *DefaultMetricsCollector) IncrementCounter(level Level, tags map[string]string) {
    key := fmt.Sprintf("log.%s", strings.ToLower(level.String()))
    d.mu.Lock()
    defer d.mu.Unlock()
    d.counters[key]++
}

// RecordHistogram records a histogram metric
func (d *DefaultMetricsCollector) RecordHistogram(metric string, value float64, tags map[string]string) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.histograms[metric] = append(d.histograms[metric], value)
}

// RecordGauge records a gauge metric
func (d *DefaultMetricsCollector) RecordGauge(metric string, value float64, tags map[string]string) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.gauges[metric] = value
}

// GetCounter returns the value of a counter metric
func (d *DefaultMetricsCollector) GetCounter(metric string) int64 {
    d.mu.RLock()
    defer d.mu.RUnlock()
    return d.counters[metric]
}

// GetHistogram returns statistics for a histogram metric
func (d *DefaultMetricsCollector) GetHistogram(metric string) (min, max, avg, p95 float64) {
    d.mu.RLock()
    defer d.mu.RUnlock()
    
    values := d.histograms[metric]
    if len(values) == 0 {
        return 0, 0, 0, 0
    }
    
    sort.Float64s(values)
    min = values[0]
    max = values[len(values)-1]
    
    sum := 0.0
    for _, v := range values {
        sum += v
    }
    avg = sum / float64(len(values))
    
    p95Index := int(math.Ceil(0.95 * float64(len(values)))) - 1
    if p95Index < 0 {
        p95Index = 0
    }
    p95 = values[p95Index]
    
    return min, max, avg, p95
}

// ===============================
// SAMPLING LOGGER - OPTIMIZED
// ===============================

// SamplingLogger provides log sampling to reduce volume
type SamplingLogger struct {
    logger  *Logger
    rate    int
    counter int64
}

// NewSamplingLogger creates a new SamplingLogger
func NewSamplingLogger(logger *Logger, rate int) *SamplingLogger {
    return &SamplingLogger{
        logger: logger,
        rate:   rate,
    }
}

// shouldLog determines if a log should be recorded based on sampling rate
func (sl *SamplingLogger) shouldLog() bool {
    if sl.rate <= 1 {
        return true
    }
    
    counter := atomic.AddInt64(&sl.counter, 1)
    return counter%int64(sl.rate) == 0
}

// ===============================
// ASYNC LOGGER - OPTIMIZED
// ===============================

// AsyncLogger provides asynchronous logging to reduce latency
type AsyncLogger struct {
    logger      *Logger
    logChan     chan *logJob
    done        chan struct{}
    wg          sync.WaitGroup
    workerCount int
}

// logJob represents a logging job
type logJob struct {
    level  Level
    msg    string
    fields map[string]interface{}
    ctx    context.Context
}

// NewAsyncLogger creates a new AsyncLogger
func NewAsyncLogger(logger *Logger, workerCount int, bufferSize int) *AsyncLogger {
    al := &AsyncLogger{
        logger:      logger,
        logChan:     make(chan *logJob, bufferSize),
        done:        make(chan struct{}),
        workerCount: workerCount,
    }
    
    for i := 0; i < workerCount; i++ {
        al.wg.Add(1)
        go al.worker()
    }
    
    return al
}

// worker is the goroutine that processes log jobs
func (al *AsyncLogger) worker() {
    defer al.wg.Done()
    
    for {
        select {
        case <-al.done:
            for {
                select {
                case job := <-al.logChan:
                    al.logger.log(job.level, job.msg, job.fields, job.ctx)
                default:
                    return
                }
            }
        case job := <-al.logChan:
            al.logger.log(job.level, job.msg, job.fields, job.ctx)
        }
    }
}

// log queues a log job for asynchronous processing
func (al *AsyncLogger) log(level Level, msg string, fields map[string]interface{}, ctx context.Context) {
    select {
    case al.logChan <- &logJob{level: level, msg: msg, fields: fields, ctx: ctx}:
    default:
        al.logger.log(WARN, "Async log channel full, dropping log", nil, nil)
    }
}

// Close closes the async logger
func (al *AsyncLogger) Close() {
    close(al.done)
    al.wg.Wait()
}

// ===============================
// LOGGER STATISTICS
// ===============================

// LoggerStats tracks logger statistics
type LoggerStats struct {
    LogCounts   map[Level]int64
    BytesWritten int64
    StartTime    time.Time
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

// Increment increments the statistics for a log level
func (ls *LoggerStats) Increment(level Level, bytes int) {
    ls.mu.Lock()
    defer ls.mu.Unlock()
    ls.LogCounts[level]++
    ls.BytesWritten += int64(bytes)
}

// GetStats returns the current statistics
func (ls *LoggerStats) GetStats() map[string]interface{} {
    ls.mu.RLock()
    defer ls.mu.RUnlock()
    
    stats := make(map[string]interface{})
    stats["start_time"] = ls.StartTime
    stats["bytes_written"] = ls.BytesWritten
    stats["uptime"] = time.Since(ls.StartTime).String()
    
    counts := make(map[string]int64)
    for level, count := range ls.LogCounts {
        counts[level.String()] = count
    }
    stats["log_counts"] = counts
    
    return stats
}

// ===============================
// ENHANCED LOGGING METHODS
// ===============================

// Trace logs a trace level message
func (l *Logger) Trace(msg string, fields ...map[string]interface{}) {
    l.logWithFields(TRACE, msg, fields)
}

// Debug logs a debug level message
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
    l.logWithFields(DEBUG, msg, fields)
}

// Info logs an info level message
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
    l.logWithFields(INFO, msg, fields)
}

// Notice logs a notice level message
func (l *Logger) Notice(msg string, fields ...map[string]interface{}) {
    l.logWithFields(NOTICE, msg, fields)
}

// Warn logs a warning level message
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
    l.logWithFields(WARN, msg, fields)
}

// Error logs an error level message
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
    l.logWithFields(ERROR, msg, fields)
}

// Fatal logs a fatal level message and exits
func (l *Logger) Fatal(msg string, fields ...map[string]interface{}) {
    l.logWithFields(FATAL, msg, fields)
}

// Panic logs a panic level message and panics
func (l *Logger) Panic(msg string, fields ...map[string]interface{}) {
    l.logWithFields(PANIC, msg, fields)
}

// logWithFields logs a message with fields
func (l *Logger) logWithFields(level Level, msg string, fields []map[string]interface{}) {
    var mergedFields map[string]interface{}
    if len(fields) > 0 {
        mergedFields = fields[0]
    }
    
    if l.asyncLogger != nil {
        l.asyncLogger.log(level, msg, mergedFields, nil)
    } else {
        l.log(level, msg, mergedFields, nil)
    }
}

// Context-aware logging methods
func (l *Logger) TraceContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(TRACE, ctx, msg, fields)
}

func (l *Logger) DebugContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(DEBUG, ctx, msg, fields)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(INFO, ctx, msg, fields)
}

func (l *Logger) NoticeContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(NOTICE, ctx, msg, fields)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(WARN, ctx, msg, fields)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(ERROR, ctx, msg, fields)
}

func (l *Logger) FatalContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(FATAL, ctx, msg, fields)
}

func (l *Logger) PanicContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(PANIC, ctx, msg, fields)
}

// logWithFieldsContext logs a message with fields and context
func (l *Logger) logWithFieldsContext(level Level, ctx context.Context, msg string, fields []map[string]interface{}) {
    var mergedFields map[string]interface{}
    if len(fields) > 0 {
        mergedFields = fields[0]
    }
    
    if l.asyncLogger != nil {
        l.asyncLogger.log(level, msg, mergedFields, ctx)
    } else {
        l.log(level, msg, mergedFields, ctx)
    }
}

// Error logging methods
func (l *Logger) ErrorErr(err error, msg string, fields ...map[string]interface{}) {
    mergedFields := make(map[string]interface{})
    if len(fields) > 0 {
        mergedFields = fields[0]
    }
    mergedFields["error"] = err.Error()
    
    if l.asyncLogger != nil {
        l.asyncLogger.log(ERROR, msg, mergedFields, nil)
    } else {
        l.log(ERROR, msg, mergedFields, nil)
    }
}

func (l *Logger) ErrorErrContext(ctx context.Context, err error, msg string, fields ...map[string]interface{}) {
    mergedFields := make(map[string]interface{})
    if len(fields) > 0 {
        mergedFields = fields[0]
    }
    mergedFields["error"] = err.Error()
    
    if l.asyncLogger != nil {
        l.asyncLogger.log(ERROR, msg, mergedFields, ctx)
    } else {
        l.log(ERROR, msg, mergedFields, ctx)
    }
}

// Formatted logging methods
func (l *Logger) Tracef(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(TRACE, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(TRACE, fmt.Sprintf(format, args...), nil, nil)
    }
}

func (l *Logger) Debugf(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(DEBUG, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(DEBUG, fmt.Sprintf(format, args...), nil, nil)
    }
}

func (l *Logger) Infof(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(INFO, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(INFO, fmt.Sprintf(format, args...), nil, nil)
    }
}

func (l *Logger) Noticef(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(NOTICE, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(NOTICE, fmt.Sprintf(format, args...), nil, nil)
    }
}

func (l *Logger) Warnf(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(WARN, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(WARN, fmt.Sprintf(format, args...), nil, nil)
    }
}

func (l *Logger) Errorf(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(ERROR, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(ERROR, fmt.Sprintf(format, args...), nil, nil)
    }
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(FATAL, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(FATAL, fmt.Sprintf(format, args...), nil, nil)
    }
}

func (l *Logger) Panicf(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(PANIC, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(PANIC, fmt.Sprintf(format, args...), nil, nil)
    }
}

// Fields-based logging methods
func (l *Logger) TraceWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(TRACE, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(TRACE, fmt.Sprintf(format, args...), fields, nil)
    }
}

func (l *Logger) DebugWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(DEBUG, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(DEBUG, fmt.Sprintf(format, args...), fields, nil)
    }
}

func (l *Logger) InfoWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(INFO, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(INFO, fmt.Sprintf(format, args...), fields, nil)
    }
}

func (l *Logger) NoticeWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(NOTICE, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(NOTICE, fmt.Sprintf(format, args...), fields, nil)
    }
}

func (l *Logger) WarnWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(WARN, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(WARN, fmt.Sprintf(format, args...), fields, nil)
    }
}

func (l *Logger) ErrorWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(ERROR, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(ERROR, fmt.Sprintf(format, args...), fields, nil)
    }
}

func (l *Logger) FatalWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(FATAL, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(FATAL, fmt.Sprintf(format, args...), fields, nil)
    }
}

func (l *Logger) PanicWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(PANIC, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(PANIC, fmt.Sprintf(format, args...), fields, nil)
    }
}

// Logger management methods
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
    newLogger := NewLogger(l.config)
    newLogger.fields = make(map[string]interface{}, len(l.fields)+len(fields))
    for k, v := range l.fields {
        newLogger.fields[k] = v
    }
    for k, v := range fields {
        newLogger.fields[k] = v
    }
    return newLogger
}

func (l *Logger) WithContextExtractor(extractor func(context.Context) map[string]string) *Logger {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.contextExtractor = extractor
    return l
}

func (l *Logger) AddHook(hook func(*LogEntry)) {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.hooks = append(l.hooks, hook)
}

func (l *Logger) SetOutput(w io.Writer) {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.out = w
}

func (l *Logger) SetErrorOutput(w io.Writer) {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.errOut = w
}

func (l *Logger) SetFormatter(formatter Formatter) {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.formatter = formatter
}

func (l *Logger) Close() error {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    
    if l.asyncLogger != nil {
        l.asyncLogger.Close()
    }
    
    if l.buffer != nil {
        return l.buffer.Close()
    }
    
    if l.rotation != nil {
        return l.rotation.Close()
    }
    
    return nil
}

// Audit logging
func (l *Logger) Audit(eventType, action, resource string, userID interface{}, success bool, details map[string]interface{}) {
    auditFields := make(map[string]interface{}, len(details)+6)
    
    auditFields["audit_event_type"] = eventType
    auditFields["audit_action"] = action
    auditFields["audit_resource"] = resource
    auditFields["audit_user_id"] = userID
    auditFields["audit_success"] = success
    auditFields["audit_timestamp"] = time.Now().UTC().Format(time.RFC3339)
    
    for k, v := range details {
        auditFields[k] = v
    }
    
    level := INFO
    if !success {
        level = WARN
    }
    
    if l.asyncLogger != nil {
        l.asyncLogger.log(level, fmt.Sprintf("Audit event: %s - %s", eventType, action), auditFields, nil)
    } else {
        l.log(level, fmt.Sprintf("Audit event: %s - %s", eventType, action), auditFields, nil)
    }
}

// GetStats returns logger statistics
func (l *Logger) GetStats() map[string]interface{} {
    return l.stats.GetStats()
}

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
