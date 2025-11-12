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
)

// ===============================
// LEVEL DEFINITION
// ===============================
// Level represents the severity level of a log entry
// Level merepresentasikan tingkat keparahan entri log
type Level int

const (
    // TRACE level for very detailed debugging information
    // Level TRACE untuk informasi debugging yang sangat detail
    TRACE Level = iota
    
    // DEBUG level for debugging information
    // Level DEBUG untuk informasi debugging
    DEBUG
    
    // INFO level for general information messages
    // Level INFO untuk pesan informasi umum
    INFO
    
    // NOTICE level for normal but significant conditions
    // Level NOTICE untuk kondisi normal namun signifikan
    NOTICE
    
    // WARN level for warning messages
    // Level WARN untuk pesan peringatan
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

var (
    // String representations of log levels
    // Representasi string dari level log
    levelStrings = []string{
        "TRACE",
        "DEBUG",
        "INFO",
        "NOTICE",
        "WARN",
        "ERROR",
        "FATAL",
        "PANIC",
    }
    
    // ANSI color codes for each log level
    // Kode warna ANSI untuk setiap level log
    levelColors = []string{
        "\033[38;5;246m",    // Gray - TRACE
        "\033[36m",          // Cyan - DEBUG
        "\033[32m",          // Green - INFO
        "\033[38;5;220m",    // Yellow - NOTICE
        "\033[33m",          // Orange - WARN
        "\033[31m",          // Red - ERROR
        "\033[38;5;198m",    // Magenta - FATAL
        "\033[38;5;196m",    // Bright Red - PANIC
    }
    
    // ANSI background color codes for each log level
    // Kode warna latar belakang ANSI untuk setiap level log
    levelBackgrounds = []string{
        "\033[48;5;238m",    // Dark gray background
        "\033[48;5;236m",
        "\033[48;5;28m",
        "\033[48;5;94m",
        "\033[48;5;130m",
        "\033[48;5;88m",
        "\033[48;5;90m",
        "\033[48;5;52m",
    }
)

// String returns the string representation of the level
// String mengembalikan representasi string dari level
func (l Level) String() string {
    if l >= TRACE && l <= PANIC {
        return levelStrings[l]
    }
    return "UNKNOWN"
}

// ParseLevel parses level from string
// ParseLevel mem-parsing level dari string
func ParseLevel(levelStr string) (Level, error) {
    switch strings.ToUpper(levelStr) {
    case "TRACE":
        return TRACE, nil
    case "DEBUG":
        return DEBUG, nil
    case "INFO":
        return INFO, nil
    case "NOTICE":
        return NOTICE, nil
    case "WARN", "WARNING":
        return WARN, nil
    case "ERROR":
        return ERROR, nil
    case "FATAL":
        return FATAL, nil
    case "PANIC":
        return PANIC, nil
    default:
        return INFO, fmt.Errorf("invalid log level: %s", levelStr)
    }
}

// ===============================
// LOG ENTRY STRUCT - OPTIMIZED VERSION
// ===============================
// LogEntry represents a single log entry with all its metadata
// LogEntry merepresentasikan entri log tunggal dengan semua metadata-nya
type LogEntry struct {
    Timestamp     time.Time              `json:"timestamp"`     // When the log was created
    Level         Level                  `json:"level"`         // Log severity level
    LevelName     string                 `json:"level_name"`     // String representation of level
    Message       string                 `json:"message"`        // Log message
    Caller        *CallerInfo            `json:"caller,omitempty"`        // Caller information
    Fields        map[string]interface{} `json:"fields,omitempty"`        // Additional fields
    PID           int                    `json:"pid"`           // Process ID
    GoroutineID   string                 `json:"goroutine_id,omitempty"`   // Goroutine ID
    TraceID       string                 `json:"trace_id,omitempty"`       // Trace ID for distributed tracing
    SpanID        string                 `json:"span_id,omitempty"`        // Span ID for distributed tracing
    UserID        string                 `json:"user_id,omitempty"`        // User ID
    SessionID     string                 `json:"session_id,omitempty"`     // Session ID
    RequestID     string                 `json:"request_id,omitempty"`     // Request ID
    Duration      time.Duration          `json:"duration,omitempty"`       // Operation duration
    Error         error                  `json:"error,omitempty"`         // Error information
    StackTrace    string                 `json:"stack_trace,omitempty"`    // Stack trace
    Hostname      string                 `json:"hostname,omitempty"`      // Hostname
    Application   string                 `json:"application,omitempty"`   // Application name
    Version       string                 `json:"version,omitempty"`       // Application version
    Environment   string                 `json:"environment,omitempty"`   // Environment (dev/prod/etc)
    CustomMetrics map[string]float64     `json:"custom_metrics,omitempty"` // Custom metrics
    Tags          []string               `json:"tags,omitempty"`          // Tags for categorization
}

// CallerInfo contains information about the code location where the log was created
// CallerInfo berisi informasi tentang lokasi kode di mana log dibuat
type CallerInfo struct {
    File     string `json:"file"`     // Source file name
    Line     int    `json:"line"`     // Line number
    Function string `json:"function"` // Function name
    Package  string `json:"package"`  // Package name
}

// Object pool for reusing LogEntry objects to reduce memory allocation
// Pool objek untuk menggunakan kembali objek LogEntry mengurangi alokasi memori
var entryPool = sync.Pool{
    New: func() interface{} {
        return &LogEntry{
            Fields:        make(map[string]interface{}),
            CustomMetrics: make(map[string]float64),
            Tags:          make([]string, 0),
        }
    },
}

// Get a LogEntry from the pool
// Mendapatkan LogEntry dari pool
func getEntryFromPool() *LogEntry {
    entry := entryPool.Get().(*LogEntry)
    // Reset fields to avoid data leakage
    // Reset field untuk menghindari kebocoran data
    entry.Timestamp = time.Time{}
    entry.Level = INFO
    entry.LevelName = ""
    entry.Message = ""
    entry.Caller = nil
    for k := range entry.Fields {
        delete(entry.Fields, k)
    }
    entry.PID = 0
    entry.GoroutineID = ""
    entry.TraceID = ""
    entry.SpanID = ""
    entry.UserID = ""
    entry.SessionID = ""
    entry.RequestID = ""
    entry.Duration = 0
    entry.Error = nil
    entry.StackTrace = ""
    entry.Hostname = ""
    entry.Application = ""
    entry.Version = ""
    entry.Environment = ""
    for k := range entry.CustomMetrics {
        delete(entry.CustomMetrics, k)
    }
    entry.Tags = entry.Tags[:0]
    return entry
}

// Return a LogEntry to the pool
// Mengembalikan LogEntry ke pool
func putEntryToPool(entry *LogEntry) {
    entryPool.Put(entry)
}

// ===============================
// CONTEXT SUPPORT - OPTIMIZED VERSION
// ===============================
// contextKey is a type for context keys to avoid collisions
// contextKey adalah tipe untuk kunci konteks untuk menghindari tabrakan
type contextKey string

const (
    // TraceIDKey is the context key for trace ID
    // TraceIDKey adalah kunci konteks untuk trace ID
    TraceIDKey contextKey = "trace_id"
    
    // SpanIDKey is the context key for span ID
    // SpanIDKey adalah kunci konteks untuk span ID
    SpanIDKey contextKey = "span_id"
    
    // UserIDKey is the context key for user ID
    // UserIDKey adalah kunci konteks untuk user ID
    UserIDKey contextKey = "user_id"
    
    // SessionIDKey is the context key for session ID
    // SessionIDKey adalah kunci konteks untuk session ID
    SessionIDKey contextKey = "session_id"
    
    // RequestIDKey is the context key for request ID
    // RequestIDKey adalah kunci konteks untuk request ID
    RequestIDKey contextKey = "request_id"
    
    // ClientIPKey is the context key for client IP
    // ClientIPKey adalah kunci konteks untuk IP klien
    ClientIPKey contextKey = "client_ip"
)

// WithTraceID adds trace ID to context
// WithTraceID menambahkan trace ID ke konteks
func WithTraceID(ctx context.Context, traceID string) context.Context {
    return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithSpanID adds span ID to context
// WithSpanID menambahkan span ID ke konteks
func WithSpanID(ctx context.Context, spanID string) context.Context {
    return context.WithValue(ctx, SpanIDKey, spanID)
}

// WithUserID adds user ID to context
// WithUserID menambahkan user ID ke konteks
func WithUserID(ctx context.Context, userID string) context.Context {
    return context.WithValue(ctx, UserIDKey, userID)
}

// WithSessionID adds session ID to context
// WithSessionID menambahkan session ID ke konteks
func WithSessionID(ctx context.Context, sessionID string) context.Context {
    return context.WithValue(ctx, SessionIDKey, sessionID)
}

// WithRequestID adds request ID to context
// WithRequestID menambahkan request ID ke konteks
func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, RequestIDKey, requestID)
}

// ExtractFromContext extracts all context values - Optimized version
// ExtractFromContext mengekstrak semua nilai konteks - Versi optimal
func ExtractFromContext(ctx context.Context) map[string]string {
    // Pre-allocate capacity to reduce allocations
    // Pra-alokasi kapasitas untuk mengurangi alokasi
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
// ADVANCED FORMATTERS - OPTIMIZED VERSION
// ===============================
// Formatter interface defines how log entries are formatted
// Interface Formatter mendefinisikan bagaimana entri log diformat
type Formatter interface {
    // Format formats a log entry into a byte slice
    // Format memformat entri log menjadi slice byte
    Format(entry *LogEntry) ([]byte, error)
}

// Byte buffer pool for reusing buffers to reduce memory allocation
// Pool buffer byte untuk menggunakan kembali buffer mengurangi alokasi memori
var bufferPool = sync.Pool{
    New: func() interface{} {
        return bytes.NewBuffer(make([]byte, 0, 1024))
    },
}

// Get a byte buffer from the pool
// Mendapatkan buffer byte dari pool
func getBufferFromPool() *bytes.Buffer {
    buf := bufferPool.Get().(*bytes.Buffer)
    buf.Reset()
    return buf
}

// Return a byte buffer to the pool
// Mengembalikan buffer byte ke pool
func putBufferToPool(buf *bytes.Buffer) {
    // Avoid keeping overly large buffers in the pool
    // Hindari menyimpan buffer yang terlalu besar di pool
    if buf.Cap() < 64*1024 {
        bufferPool.Put(buf)
    }
}

// TextFormatter - Optimized version - Returns byte slice instead of string to reduce memory allocation
// TextFormatter - Versi optimal - Mengembalikan slice byte bukan string untuk mengurangi alokasi memori
type TextFormatter struct {
    EnableColors          bool     // Enable ANSI colors in output
    ShowTimestamp         bool     // Show timestamp in output
    ShowCaller            bool     // Show caller information
    ShowGoroutine         bool     // Show goroutine ID
    ShowPID               bool     // Show process ID
    ShowTraceInfo         bool     // Show trace information
    ShowHostname          bool     // Show hostname
    ShowApplication       bool     // Show application name
    FullTimestamp         bool     // Show full timestamp with nanoseconds
    TimestampFormat       string   // Custom timestamp format
    IndentFields          bool     // Indent fields for better readability
    MaxFieldWidth         int      // Maximum width for field values
    EnableStackTrace      bool     // Enable stack trace for errors
    StackTraceDepth       int      // Maximum stack trace depth
    EnableDuration        bool     // Show operation duration
    CustomFieldOrder      []string // Custom order for fields
    EnableColorsByLevel   bool     // Enable colors based on log level
    FieldTransformers     map[string]func(interface{}) string // Functions to transform field values
    SensitiveFields       []string // List of sensitive field names
    MaskSensitiveData     bool     // Whether to mask sensitive data
    MaskString            string   // String to use for masking
}

// Format formats a log entry into a byte slice
// Format memformat entri log menjadi slice byte
func (f *TextFormatter) Format(entry *LogEntry) ([]byte, error) {
    // Get buffer from pool to reduce allocations
    // Mendapatkan buffer dari pool untuk mengurangi alokasi
    buf := getBufferFromPool()
    defer putBufferToPool(buf)
    
    // Use more efficient write methods
    // Menggunakan metode tulis yang lebih efisien
    writeByte := func(b byte) {
        buf.WriteByte(b)
    }
    
    writeString := func(s string) {
        buf.WriteString(s)
    }
    
    // Write timestamp
    // Menulis timestamp
    if f.ShowTimestamp {
        writeByte('[')
        writeString(entry.Timestamp.Format(f.TimestampFormat))
        writeByte(']')
        writeByte(' ')
    }
    
    // Write level with background and padding
    // Menulis level dengan latar belakang dan padding
    levelStr := levelStrings[entry.Level]
    if f.EnableColors {
        writeString(levelBackgrounds[entry.Level])
        writeString(levelColors[entry.Level])
        writeByte(' ')
        writeString(levelStr)
        writeByte(' ')
        writeString("\033[0m")
    } else {
        writeByte('[')
        writeString(levelStr)
        writeByte(']')
    }
    writeByte(' ')
    
    // Write hostname
    // Menulis hostname
    if f.ShowHostname && entry.Hostname != "" {
        if f.EnableColors {
            writeString("\033[38;5;245m")
        }
        writeString(entry.Hostname)
        if f.EnableColors {
            writeString("\033[0m")
        }
        writeByte(' ')
    }
    
    // Write application name
    // Menulis nama aplikasi
    if f.ShowApplication && entry.Application != "" {
        if f.EnableColors {
            writeString("\033[38;5;245m")
        }
        writeString(entry.Application)
        if f.EnableColors {
            writeString("\033[0m")
        }
        writeByte(' ')
    }
    
    // Write process ID
    // Menulis ID proses
    if f.ShowPID {
        if f.EnableColors {
            writeString("\033[38;5;245m")
        }
        writeString("PID:")
        writeString(strconv.Itoa(entry.PID))
        if f.EnableColors {
            writeString("\033[0m")
        }
        writeByte(' ')
    }
    
    // Write goroutine ID
    // Menulis ID goroutine
    if f.ShowGoroutine && entry.GoroutineID != "" {
        if f.EnableColors {
            writeString("\033[38;5;245m")
        }
        writeString("GID:")
        writeString(entry.GoroutineID)
        if f.EnableColors {
            writeString("\033[0m")
        }
        writeByte(' ')
    }
    
    // Write trace information
    // Menulis informasi trace
    if f.ShowTraceInfo {
        if entry.TraceID != "" {
            if f.EnableColors {
                writeString("\033[38;5;141m") // Purple
            }
            writeString("TRACE:")
            writeString(shortID(entry.TraceID))
            if f.EnableColors {
                writeString("\033[0m")
            }
            writeByte(' ')
        }
        
        if entry.SpanID != "" {
            if f.EnableColors {
                writeString("\033[38;5;141m") // Purple
            }
            writeString("SPAN:")
            writeString(shortID(entry.SpanID))
            if f.EnableColors {
                writeString("\033[0m")
            }
            writeByte(' ')
        }
        
        if entry.RequestID != "" {
            if f.EnableColors {
                writeString("\033[38;5;141m") // Purple
            }
            writeString("REQ:")
            writeString(shortID(entry.RequestID))
            if f.EnableColors {
                writeString("\033[0m")
            }
            writeByte(' ')
        }
    }
    
    // Write caller information
    // Menulis informasi pemanggil
    if f.ShowCaller && entry.Caller != nil {
        if f.EnableColors {
            writeString("\033[38;5;246m")
        }
        writeString(entry.Caller.File)
        writeByte(':')
        writeString(strconv.Itoa(entry.Caller.Line))
        if f.EnableColors {
            writeString("\033[0m")
        }
        writeByte(' ')
    }
    
    // Write duration
    // Menulis durasi
    if f.EnableDuration && entry.Duration > 0 {
        if f.EnableColors {
            writeString("\033[38;5;155m") // Light green
        }
        writeByte('(')
        writeString(entry.Duration.String())
        writeByte(')')
        if f.EnableColors {
            writeString("\033[0m")
        }
        writeByte(' ')
    }
    
    // Write message with color based on level
    // Menulis pesan dengan warna berdasarkan level
    if f.EnableColors && f.EnableColorsByLevel {
        writeString(levelColors[entry.Level])
        if entry.Level >= ERROR {
            writeString("\033[1m") // Bold for important messages
        }
    }
    writeString(entry.Message)
    if f.EnableColors && f.EnableColorsByLevel {
        writeString("\033[0m")
    }
    
    // Write error
    // Menulis kesalahan
    if entry.Error != nil {
        writeByte(' ')
        if f.EnableColors {
            writeString("\033[38;5;196m") // Bright red for errors
        }
        writeString("error=\"")
        writeString(entry.Error.Error())
        writeByte('"')
        if f.EnableColors {
            writeString("\033[0m")
        }
    }
    
    // Write fields
    // Menulis field
    if len(entry.Fields) > 0 {
        writeByte(' ')
        f.formatFields(buf, entry.Fields)
    }
    
    // Write tags
    // Menulis tag
    if len(entry.Tags) > 0 {
        writeByte(' ')
        f.formatTags(buf, entry.Tags)
    }
    
    // Write metrics
    // Menulis metrik
    if len(entry.CustomMetrics) > 0 {
        writeByte(' ')
        f.formatMetrics(buf, entry.CustomMetrics)
    }
    
    // Write stack trace
    // Menulis stack trace
    if f.EnableStackTrace && entry.StackTrace != "" {
        writeByte('\n')
        if f.EnableColors {
            writeString("\033[38;5;240m") // Dark gray for stack trace
        }
        writeString(entry.StackTrace)
        if f.EnableColors {
            writeString("\033[0m")
        }
    }
    
    writeByte('\n')
    
    // Return a copy of the byte slice to avoid buffer reuse issues
    // Mengembalikan salinan slice byte untuk menghindari masalah penggunaan ulang buffer
    result := make([]byte, buf.Len())
    copy(result, buf.Bytes())
    return result, nil
}

// formatFields formats the fields map into the buffer
// formatFields memformat map field ke dalam buffer
func (f *TextFormatter) formatFields(buf *bytes.Buffer, fields map[string]interface{}) {
    if f.EnableColors {
        buf.WriteString("\033[38;5;243m")
    }
    buf.WriteByte('{')
    
    // Use custom field order if specified
    // Menggunakan urutan field kustom jika ditentukan
    keys := make([]string, 0, len(fields))
    if len(f.CustomFieldOrder) > 0 {
        // Add ordered fields first
        // Menambahkan field yang diurutkan terlebih dahulu
        for _, key := range f.CustomFieldOrder {
            if _, exists := fields[key]; exists {
                keys = append(keys, key)
            }
        }
        // Add remaining fields
        // Menambahkan field yang tersisa
        for key := range fields {
            if !contains(f.CustomFieldOrder, key) {
                keys = append(keys, key)
            }
        }
    } else {
        for key := range fields {
            keys = append(keys, key)
        }
    }
    
    for i, k := range keys {
        v := fields[k]
        if i > 0 {
            buf.WriteByte(' ')
        }
        
        if f.EnableColors {
            buf.WriteString("\033[38;5;228m") // Light yellow for keys
        }
        buf.WriteString(k)
        buf.WriteByte('=')
        if f.EnableColors {
            buf.WriteString("\033[38;5;159m") // Light cyan for values
        }
        
        // Apply field transformer if exists
        // Menerapkan transformer field jika ada
        if transformer, exists := f.FieldTransformers[k]; exists {
            buf.WriteString(transformer(v))
        } else {
            // Mask sensitive data
            // Menyembunyikan data sensitif
            if f.MaskSensitiveData && contains(f.SensitiveFields, k) {
                buf.WriteString(f.MaskString)
            } else {
                strVal := formatValue(v, f.MaxFieldWidth)
                buf.WriteString(strVal)
            }
        }
    }
    
    if f.EnableColors {
        buf.WriteString("\033[38;5;243m")
    }
    buf.WriteByte('}')
    if f.EnableColors {
        buf.WriteString("\033[0m")
    }
}

// formatTags formats the tags slice into the buffer
// formatTags memformat slice tag ke dalam buffer
func (f *TextFormatter) formatTags(buf *bytes.Buffer, tags []string) {
    if f.EnableColors {
        buf.WriteString("\033[38;5;135m") // Purple for tags
    }
    buf.WriteByte('[')
    for i, tag := range tags {
        if i > 0 {
            buf.WriteByte(',')
        }
        buf.WriteString(tag)
    }
    buf.WriteByte(']')
    if f.EnableColors {
        buf.WriteString("\033[0m")
    }
}

// formatMetrics formats the metrics map into the buffer
// formatMetrics memformat map metrik ke dalam buffer
func (f *TextFormatter) formatMetrics(buf *bytes.Buffer, metrics map[string]float64) {
    if f.EnableColors {
        buf.WriteString("\033[38;5;85m") // Green for metrics
    }
    buf.WriteByte('<')
    first := true
    for k, v := range metrics {
        if !first {
            buf.WriteByte(' ')
        }
        first = false
        buf.WriteString(k)
        buf.WriteByte('=')
        buf.WriteString(strconv.FormatFloat(v, 'f', 2, 64))
    }
    buf.WriteByte('>')
    if f.EnableColors {
        buf.WriteString("\033[0m")
    }
}

// JSONFormatter - Optimized version - Returns byte slice instead of string to reduce memory allocation
// JSONFormatter - Versi optimal - Mengembalikan slice byte bukan string untuk mengurangi alokasi memori
type JSONFormatter struct {
    PrettyPrint          bool     // Enable pretty-printed JSON
    TimestampFormat      string   // Custom timestamp format
    ShowCaller           bool     // Show caller information
    ShowGoroutine        bool     // Show goroutine ID
    ShowPID              bool     // Show process ID
    ShowTraceInfo        bool     // Show trace information
    EnableStackTrace     bool     // Enable stack trace for errors
    EnableDuration       bool     // Show operation duration
    FieldKeyMap          map[string]string // Map for renaming fields
    DisableHTMLEscape    bool     // Disable HTML escaping in JSON
    SensitiveFields      []string // List of sensitive field names
    MaskSensitiveData    bool     // Whether to mask sensitive data
    MaskString           string   // String to use for masking
    FieldTransformers    map[string]func(interface{}) interface{} // Functions to transform field values
}

// NewJSONFormatter creates a new JSONFormatter
// NewJSONFormatter membuat JSONFormatter baru
func NewJSONFormatter() *JSONFormatter {
    return &JSONFormatter{}
}

// Format formats a log entry into JSON byte slice
// Format memformat entri log menjadi slice byte JSON
func (f *JSONFormatter) Format(entry *LogEntry) ([]byte, error) {
    // Create a copy to avoid modifying the original
    // Membuat salinan untuk menghindari modifikasi asli
    outputEntry := &LogEntry{
        Timestamp:   entry.Timestamp,
        Level:       entry.Level,
        LevelName:   entry.LevelName,
        Message:     entry.Message,
        PID:         entry.PID,
        Error:       entry.Error,
        Hostname:    entry.Hostname,
        Application: entry.Application,
        Version:     entry.Version,
        Environment: entry.Environment,
    }
    
    if f.ShowCaller {
        outputEntry.Caller = entry.Caller
    }
    
    if f.ShowGoroutine {
        outputEntry.GoroutineID = entry.GoroutineID
    }
    
    if f.ShowPID {
        outputEntry.PID = entry.PID
    }
    
    if f.ShowTraceInfo {
        outputEntry.TraceID = entry.TraceID
        outputEntry.SpanID = entry.SpanID
        outputEntry.UserID = entry.UserID
        outputEntry.SessionID = entry.SessionID
        outputEntry.RequestID = entry.RequestID
    }
    
    if f.EnableDuration {
        outputEntry.Duration = entry.Duration
    }
    
    if f.EnableStackTrace {
        outputEntry.StackTrace = entry.StackTrace
    }
    
    if len(entry.Fields) > 0 {
        outputEntry.Fields = f.processFields(entry.Fields)
    }
    
    if len(entry.Tags) > 0 {
        outputEntry.Tags = entry.Tags
    }
    
    if len(entry.CustomMetrics) > 0 {
        outputEntry.CustomMetrics = entry.CustomMetrics
    }
    
    // Format timestamp if needed
    // Format timestamp jika diperlukan
    if f.TimestampFormat != "" {
        outputEntry.Timestamp = entry.Timestamp
    }
    
    var data []byte
    var err error
    
    if f.PrettyPrint {
        data, err = json.MarshalIndent(outputEntry, "", "  ")
    } else {
        // Create a new buffer and encoder for each call
        // Membuat buffer dan encoder baru untuk setiap panggilan
        buf := getBufferFromPool()
        defer putBufferToPool(buf)
        
        encoder := json.NewEncoder(buf)
        if f.DisableHTMLEscape {
            encoder.SetEscapeHTML(false)
        }
        err = encoder.Encode(outputEntry)
        data = buf.Bytes()
    }
    
    if err != nil {
        return nil, err
    }
    
    // Return a copy of the byte slice to avoid buffer reuse issues
    // Mengembalikan salinan slice byte untuk menghindari masalah penggunaan ulang buffer
    result := make([]byte, len(data))
    copy(result, data)
    return result, nil
}

// processFields processes and transforms fields according to configuration
// processFields memproses dan mengubah field sesuai konfigurasi
func (f *JSONFormatter) processFields(fields map[string]interface{}) map[string]interface{} {
    // Pre-allocate capacity to reduce allocations
    // Pra-alokasi kapasitas untuk mengurangi alokasi
    processed := make(map[string]interface{}, len(fields))
    
    for k, v := range fields {
        // Apply field mapping
        // Menerapkan pemetaan field
        key := k
        if mappedKey, exists := f.FieldKeyMap[k]; exists {
            key = mappedKey
        }
        
        // Apply transformer
        // Menerapkan transformer
        if transformer, exists := f.FieldTransformers[k]; exists {
            processed[key] = transformer(v)
        } else if f.MaskSensitiveData && contains(f.SensitiveFields, k) {
            processed[key] = f.MaskString
        } else {
            processed[key] = v
        }
    }
    
    return processed
}

// CSVFormatter - Optimized version - Returns byte slice instead of string to reduce memory allocation
// CSVFormatter - Versi optimal - Mengembalikan slice byte bukan string untuk mengurangi alokasi memori
type CSVFormatter struct {
    IncludeHeader    bool     // Include header row in output
    FieldOrder       []string // Order of fields in CSV
    TimestampFormat  string   // Custom timestamp format
}

// NewCSVFormatter creates a new CSVFormatter
// NewCSVFormatter membuat CSVFormatter baru
func NewCSVFormatter() *CSVFormatter {
    return &CSVFormatter{}
}

// Format formats a log entry into CSV byte slice
// Format memformat entri log menjadi slice byte CSV
func (f *CSVFormatter) Format(entry *LogEntry) ([]byte, error) {
    buf := getBufferFromPool()
    defer putBufferToPool(buf)
    
    // Create a new writer for each call
    // Membuat writer baru untuk setiap panggilan
    writer := csv.NewWriter(buf)
    
    // Create record with proper field order
    // Membuat record dengan urutan field yang benar
    record := make([]string, len(f.FieldOrder))
    for i, field := range f.FieldOrder {
        switch field {
        case "timestamp":
            record[i] = entry.Timestamp.Format(f.TimestampFormat)
        case "level":
            record[i] = entry.LevelName
        case "message":
            record[i] = entry.Message
        case "pid":
            record[i] = strconv.Itoa(entry.PID)
        case "goroutine_id":
            record[i] = entry.GoroutineID
        case "trace_id":
            record[i] = entry.TraceID
        case "file":
            if entry.Caller != nil {
                record[i] = entry.Caller.File
            }
        case "line":
            if entry.Caller != nil {
                record[i] = strconv.Itoa(entry.Caller.Line)
            }
        case "error":
            if entry.Error != nil {
                record[i] = entry.Error.Error()
            }
        default:
            if entry.Fields != nil {
                if val, exists := entry.Fields[field]; exists {
                    record[i] = fmt.Sprintf("%v", val)
                }
            }
        }
    }
    
    if err := writer.Write(record); err != nil {
        return nil, err
    }
    
    writer.Flush()
    if err := writer.Error(); err != nil {
        return nil, err
    }
    
    // Return a copy of the byte slice to avoid buffer reuse issues
    // Mengembalikan salinan slice byte untuk menghindari masalah penggunaan ulang buffer
    result := make([]byte, buf.Len())
    copy(result, buf.Bytes())
    return result, nil
}

// ===============================
// LOGGER CONFIGURATION - OPTIMIZED VERSION
// ===============================
// LoggerConfig holds configuration for the logger
// LoggerConfig menyimpan konfigurasi untuk logger
type LoggerConfig struct {
    Level              Level                  // Minimum log level to output
    EnableColors       bool                   // Enable ANSI colors in output
    Output             io.Writer              // Output writer for logs
    ErrorOutput        io.Writer              // Output writer for errors
    Formatter          Formatter              // Log formatter
    ShowCaller         bool                   // Show caller information
    CallerDepth        int                    // Depth for caller information
    ShowGoroutine      bool                   // Show goroutine ID
    ShowPID            bool                   // Show process ID
    ShowTraceInfo      bool                   // Show trace information
    ShowHostname       bool                   // Show hostname
    ShowApplication    bool                   // Show application name
    TimestampFormat    string                 // Custom timestamp format
    ExitFunc           func(int)              // Function to call on fatal
    EnableStackTrace   bool                   // Enable stack trace for errors
    StackTraceDepth    int                    // Maximum stack trace depth
    EnableSampling     bool                   // Enable log sampling
    SamplingRate       int                    // Sampling rate (1 in N logs)
    BufferSize         int                    // Buffer size for buffered writer
    FlushInterval      time.Duration          // Flush interval for buffered writer
    EnableRotation     bool                   // Enable log rotation
    RotationConfig     *RotationConfig        // Configuration for log rotation
    ContextExtractor   func(context.Context) map[string]string // Function to extract context values
    Hostname           string                 // Hostname to include in logs
    Application        string                 // Application name
    Version           string                 // Application version
    Environment        string                 // Environment (dev/prod/etc)
    MaxFieldSize       int                    // Maximum field size
    EnableMetrics      bool                   // Enable metrics collection
    MetricsCollector   MetricsCollector       // Metrics collector
    ErrorHandler       func(error)            // Error handler function
    OnFatal            func(*LogEntry)        // Callback for fatal logs
    OnPanic            func(*LogEntry)        // Callback for panic logs
    
    // New performance optimization configurations
    // Konfigurasi optimasi performa baru
    DisableLocking     bool     // Disable locks for single-threaded scenarios
    PreAllocateFields  int      // Pre-allocate field count
    PreAllocateTags    int      // Pre-allocate tag count
    MaxMessageSize     int      // Maximum message size to avoid large messages affecting performance
    AsyncLogging       bool     // Enable asynchronous logging
}

// RotationConfig holds configuration for log rotation
// RotationConfig menyimpan konfigurasi untuk rotasi log
type RotationConfig struct {
    MaxSize         int64         // Maximum file size before rotation
    MaxAge          time.Duration // Maximum age before rotation
    MaxBackups      int           // Maximum number of backup files
    LocalTime       bool          // Use local time for rotation
    Compress        bool          // Compress rotated files
    RotationTime    time.Duration // Time-based rotation interval
    FilenamePattern string        // Pattern for rotated filenames
}

// ===============================
// METRICS COLLECTOR - OPTIMIZED VERSION
// ===============================
// MetricsCollector interface defines methods for collecting metrics
// Interface MetricsCollector mendefinisikan metode untuk mengumpulkan metrik
type MetricsCollector interface {
    // IncrementCounter increments a counter metric
    // IncrementCounter menambah metrik counter
    IncrementCounter(level Level, tags map[string]string)
    
    // RecordHistogram records a histogram metric
    // RecordHistogram merekam metrik histogram
    RecordHistogram(metric string, value float64, tags map[string]string)
    
    // RecordGauge records a gauge metric
    // RecordGauge merekam metrik gauge
    RecordGauge(metric string, value float64, tags map[string]string)
}

// DefaultMetricsCollector is a simple in-memory metrics collector
// DefaultMetricsCollector adalah kolektor metrik dalam memori sederhana
type DefaultMetricsCollector struct {
    counters   map[string]int64
    histograms map[string][]float64
    gauges     map[string]float64
    mu         sync.RWMutex
}

// NewDefaultMetricsCollector creates a new DefaultMetricsCollector
// NewDefaultMetricsCollector membuat DefaultMetricsCollector baru
func NewDefaultMetricsCollector() *DefaultMetricsCollector {
    return &DefaultMetricsCollector{
        counters:   make(map[string]int64),
        histograms: make(map[string][]float64),
        gauges:     make(map[string]float64),
    }
}

// IncrementCounter increments a counter metric
// IncrementCounter menambah metrik counter
func (d *DefaultMetricsCollector) IncrementCounter(level Level, tags map[string]string) {
    key := fmt.Sprintf("log.%s", strings.ToLower(level.String()))
    d.mu.Lock()
    defer d.mu.Unlock()
    d.counters[key]++
}

// RecordHistogram records a histogram metric
// RecordHistogram merekam metrik histogram
func (d *DefaultMetricsCollector) RecordHistogram(metric string, value float64, tags map[string]string) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.histograms[metric] = append(d.histograms[metric], value)
}

// RecordGauge records a gauge metric
// RecordGauge merekam metrik gauge
func (d *DefaultMetricsCollector) RecordGauge(metric string, value float64, tags map[string]string) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.gauges[metric] = value
}

// GetCounter returns the value of a counter metric
// GetCounter mengembalikan nilai metrik counter
func (d *DefaultMetricsCollector) GetCounter(metric string) int64 {
    d.mu.RLock()
    defer d.mu.RUnlock()
    return d.counters[metric]
}

// GetHistogram returns statistics for a histogram metric
// GetHistogram mengembalikan statistik untuk metrik histogram
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
// HIGH-PERFORMANCE BUFFERED WRITER - OPTIMIZED VERSION
// ===============================
// BufferedWriter is a high-performance buffered writer with batch processing
// BufferedWriter adalah writer yang di-buffer berkinerja tinggi dengan pemrosesan batch
type BufferedWriter struct {
    writer         io.Writer         // Underlying writer
    buffer         chan []byte       // Channel for buffering log entries
    bufferSize     int              // Size of the buffer channel
    flushInterval  time.Duration    // Interval for automatic flushing
    done           chan struct{}    // Channel to signal shutdown
    wg             sync.WaitGroup   // WaitGroup for goroutine management
    mu             sync.Mutex       // Mutex for thread safety
    droppedLogs    int64            // Counter for dropped logs
    totalLogs      int64            // Counter for total logs
    lastFlush      time.Time        // Time of last flush
    
    // Optimization: Pre-allocated buffer pool
    // Optimasi: Pool buffer pra-alokasi
    bufferPool     sync.Pool        // Pool for reusing buffers
    
    // Optimization: Batch writing
    // Optimasi: Penulisan batch
    batchSize      int              // Size of batch for writing
    batchTimeout   time.Duration    // Timeout for batch writing
}

// NewBufferedWriter creates a new BufferedWriter
// NewBufferedWriter membuat BufferedWriter baru
func NewBufferedWriter(writer io.Writer, bufferSize int, flushInterval time.Duration) *BufferedWriter {
    bw := &BufferedWriter{
        writer:        writer,
        buffer:        make(chan []byte, bufferSize),
        bufferSize:    bufferSize,
        flushInterval: flushInterval,
        done:          make(chan struct{}),
        lastFlush:     time.Now(),
        batchSize:     100,  // Default batch size
        batchTimeout:  100 * time.Millisecond, // Default batch timeout
        bufferPool: sync.Pool{
            New: func() interface{} {
                return make([]byte, 0, 1024)
            },
        },
    }
    
    bw.wg.Add(1)
    go bw.flushWorker()
    
    return bw
}

// Write writes data to the buffer
// Write menulis data ke buffer
func (bw *BufferedWriter) Write(p []byte) (n int, err error) {
    atomic.AddInt64(&bw.totalLogs, 1)
    
    // Use buffer from pool to avoid repeated allocation
    // Menggunakan buffer dari pool untuk menghindari alokasi berulang
    buf := bw.bufferPool.Get().([]byte)
    if cap(buf) < len(p) {
        buf = make([]byte, len(p))
    }
    buf = buf[:len(p)]
    copy(buf, p)
    
    select {
    case bw.buffer <- buf:
        return len(p), nil
    default:
        // Buffer full, drop the log and increment counter
        // Buffer penuh, buang log dan tambahkan counter
        atomic.AddInt64(&bw.droppedLogs, 1)
        // Try direct write as fallback
        // Coba tulis langsung sebagai cadangan
        bw.mu.Lock()
        defer bw.mu.Unlock()
        return bw.writer.Write(p)
    }
}

// flushWorker is the goroutine that flushes buffered logs
// flushWorker adalah goroutine yang menyiram log yang di-buffer
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
            batch = batch[:0] // Reset batch
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

// flushBatch flushes a batch of log entries
// flushBatch menyiram batch entri log
func (bw *BufferedWriter) flushBatch(batch [][]byte) {
    if len(batch) == 0 {
        return
    }
    
    bw.mu.Lock()
    defer bw.mu.Unlock()
    
    // Optimization: Combine all buffers into one large buffer to reduce write calls
    // Optimasi: Menggabungkan semua buffer menjadi satu buffer besar untuk mengurangi panggilan tulis
    totalSize := 0
    for _, data := range batch {
        totalSize += len(data)
    }
    
    combined := make([]byte, 0, totalSize)
    for _, data := range batch {
        combined = append(combined, data...)
        // Return buffer to pool
        // Mengembalikan buffer ke pool
        bw.bufferPool.Put(data[:0])
    }
    
    bw.writer.Write(combined)
}

// Stats returns statistics about the buffered writer
// Stats mengembalikan statistik tentang writer yang di-buffer
func (bw *BufferedWriter) Stats() map[string]interface{} {
    return map[string]interface{}{
        "buffer_size":    bw.bufferSize,
        "current_queue":  len(bw.buffer),
        "dropped_logs":   atomic.LoadInt64(&bw.droppedLogs),
        "total_logs":     atomic.LoadInt64(&bw.totalLogs),
        "last_flush":     bw.lastFlush,
    }
}

// Close closes the buffered writer
// Close menutup writer yang di-buffer
func (bw *BufferedWriter) Close() error {
    close(bw.done)
    bw.wg.Wait()
    
    // Flush remaining logs
    // Menyiram log yang tersisa
    bw.flushBatch(nil) // This will flush all remaining in buffer
    
    if closer, ok := bw.writer.(io.Closer); ok {
        return closer.Close()
    }
    return nil
}

// ===============================
// HIGH-PERFORMANCE ASYNC LOGGER - OPTIMIZED VERSION
// ===============================
// AsyncLogger provides asynchronous logging to reduce latency
// AsyncLogger menyediakan logging asinkron untuk mengurangi latensi
type AsyncLogger struct {
    logger      *Logger         // Underlying logger
    logChan     chan *logJob    // Channel for log jobs
    done        chan struct{}   // Channel to signal shutdown
    wg          sync.WaitGroup  // WaitGroup for goroutine management
    workerCount int             // Number of worker goroutines
}

// logJob represents a logging job
// logJob merepresentasikan pekerjaan logging
type logJob struct {
    level  Level                  // Log level
    msg    string                 // Log message
    fields map[string]interface{} // Log fields
    ctx    context.Context        // Context
}

// NewAsyncLogger creates a new AsyncLogger
// NewAsyncLogger membuat AsyncLogger baru
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
// worker adalah goroutine yang memproses pekerjaan log
func (al *AsyncLogger) worker() {
    defer al.wg.Done()
    
    for {
        select {
        case <-al.done:
            // Process remaining log jobs
            // Memproses pekerjaan log yang tersisa
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
// log mengantrikan pekerjaan log untuk pemrosesan asinkron
func (al *AsyncLogger) log(level Level, msg string, fields map[string]interface{}, ctx context.Context) {
    select {
    case al.logChan <- &logJob{level: level, msg: msg, fields: fields, ctx: ctx}:
        // Successfully sent to channel
        // Berhasil dikirim ke channel
    default:
        // Channel full, drop log or log directly
        // Channel penuh, buang log atau log langsung
        al.logger.log(WARN, "Async log channel full, dropping log", nil, nil)
    }
}

// Close closes the async logger
// Close menutup async logger
func (al *AsyncLogger) Close() {
    close(al.done)
    al.wg.Wait()
}

// ===============================
// SAMPLING LOGGER - OPTIMIZED VERSION
// ===============================
// SamplingLogger provides log sampling to reduce volume
// SamplingLogger menyediakan sampling log untuk mengurangi volume
type SamplingLogger struct {
    logger      *Logger  // Underlying logger
    rate        int      // Sampling rate (1 in N)
    counter     int64    // Counter for sampling
    mu          sync.Mutex // Mutex for thread safety
}

// NewSamplingLogger creates a new SamplingLogger
// NewSamplingLogger membuat SamplingLogger baru
func NewSamplingLogger(logger *Logger, rate int) *SamplingLogger {
    return &SamplingLogger{
        logger: logger,
        rate:   rate,
    }
}

// shouldLog determines if a log should be recorded based on sampling rate
// shouldLog menentukan apakah log harus direkam berdasarkan tingkat sampling
func (sl *SamplingLogger) shouldLog() bool {
    if sl.rate <= 1 {
        return true
    }
    
    counter := atomic.AddInt64(&sl.counter, 1)
    return counter%int64(sl.rate) == 0
}

// ===============================
// MAIN LOGGER STRUCT - OPTIMIZED VERSION
// ===============================
// Logger is the main logging structure
// Logger adalah struktur logging utama
type Logger struct {
    config          LoggerConfig           // Logger configuration
    formatter       Formatter              // Log formatter
    out             io.Writer              // Output writer
    errOut          io.Writer              // Error output writer
    mu              sync.Mutex             // Mutex for thread safety
    hooks           []func(*LogEntry)      // Hooks for log processing
    exitFunc        func(int)              // Function to call on fatal
    fields          map[string]interface{} // Default fields
    sampler         *SamplingLogger        // Sampling logger
    buffer          *BufferedWriter        // Buffered writer
    rotation        *RotatingFileWriter    // Rotating file writer
    contextExtractor func(context.Context) map[string]string // Context extractor function
    metrics         MetricsCollector       // Metrics collector
    errorHandler    func(error)            // Error handler function
    onFatal         func(*LogEntry)        // Callback for fatal logs
    onPanic         func(*LogEntry)        // Callback for panic logs
    stats           *LoggerStats           // Logger statistics
    asyncLogger     *AsyncLogger           // Async logger
    
    // Optimization: Pre-allocated fields and tags
    // Optimasi: Field dan tag pra-alokasi
    preAllocatedFields map[string]interface{} // Pre-allocated fields
    preAllocatedTags   []string               // Pre-allocated tags
}

// LoggerStats tracks logger statistics
// LoggerStats melacak statistik logger
type LoggerStats struct {
    LogCounts      map[Level]int64 // Count of logs by level
    BytesWritten   int64           // Total bytes written
    StartTime      time.Time       // Logger start time
    mu             sync.RWMutex    // Mutex for thread safety
}

// NewLoggerStats creates a new LoggerStats
// NewLoggerStats membuat LoggerStats baru
func NewLoggerStats() *LoggerStats {
    return &LoggerStats{
        LogCounts:    make(map[Level]int64),
        BytesWritten: 0,
        StartTime:    time.Now(),
    }
}

// Increment increments the statistics for a log level
// Increment menambah statistik untuk level log
func (ls *LoggerStats) Increment(level Level, bytes int) {
    ls.mu.Lock()
    defer ls.mu.Unlock()
    ls.LogCounts[level]++
    ls.BytesWritten += int64(bytes)
}

// GetStats returns the current statistics
// GetStats mengembalikan statistik saat ini
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
// LOGGER METHODS - OPTIMIZED VERSION
// ===============================
// NewDefaultLogger creates a logger with default configuration
// NewDefaultLogger membuat logger dengan konfigurasi default
func NewDefaultLogger() *Logger {
    config := LoggerConfig{
        Level:            INFO,
        EnableColors:     true,
        Output:           os.Stdout,
        ErrorOutput:      os.Stderr,
        ShowCaller:       true,
        CallerDepth:      DEFAULT_CALLER_DEPTH,
        ShowGoroutine:    true,
        ShowPID:          true,
        ShowTraceInfo:    true,
        TimestampFormat:  DEFAULT_TIMESTAMP_FORMAT,
        EnableStackTrace: true,
        StackTraceDepth:  10,
        BufferSize:       DEFAULT_BUFFER_SIZE,
        FlushInterval:    DEFAULT_FLUSH_INTERVAL,
        PreAllocateFields: 10,
        PreAllocateTags:   5,
        MaxMessageSize:    1024 * 1024, // 1MB
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
    }
    
    return NewLogger(config)
}

// NewLogger creates a new logger with the given configuration
// NewLogger membuat logger baru dengan konfigurasi yang diberikan
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
        preAllocatedFields: make(map[string]interface{}, config.PreAllocateFields),
        preAllocatedTags:   make([]string, 0, config.PreAllocateTags),
    }
    
    if l.exitFunc == nil {
        l.exitFunc = os.Exit
    }
    
    if config.BufferSize > 0 {
        l.buffer = NewBufferedWriter(config.Output, config.BufferSize, config.FlushInterval)
        l.out = l.buffer
    }
    
    if config.EnableSampling && config.SamplingRate > 1 {
        l.sampler = NewSamplingLogger(l, config.SamplingRate)
    }
    
    if config.EnableRotation && config.RotationConfig != nil {
        // Handle rotation setup
        // Menangani pengaturan rotasi
        if file, ok := config.Output.(*os.File); ok {
            var err error
            l.rotation, err = NewRotatingFileWriter(file.Name(), config.RotationConfig)
            if err == nil {
                l.out = l.rotation
            }
        }
    }
    
    if config.AsyncLogging {
        l.asyncLogger = NewAsyncLogger(l, 4, config.BufferSize)
    }
    
    return l
}

// SetLevel sets the minimum log level
// SetLevel mengatur level log minimum
func (l *Logger) SetLevel(level Level) {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.config.Level = level
}

// GetStats returns logger statistics
// GetStats mengembalikan statistik logger
func (l *Logger) GetStats() map[string]interface{} {
    return l.stats.GetStats()
}

// Trace logs a trace level message
// Trace mencatat pesan level trace
func (l *Logger) Trace(msg string, fields ...map[string]interface{}) {
    l.logWithFields(TRACE, msg, fields)
}

// Debug logs a debug level message
// Debug mencatat pesan level debug
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
    l.logWithFields(DEBUG, msg, fields)
}

// Info logs an info level message
// Info mencatat pesan level info
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
    l.logWithFields(INFO, msg, fields)
}

// Notice logs a notice level message
// Notice mencatat pesan level notice
func (l *Logger) Notice(msg string, fields ...map[string]interface{}) {
    l.logWithFields(NOTICE, msg, fields)
}

// Warn logs a warning level message
// Warn mencatat pesan level warning
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
    l.logWithFields(WARN, msg, fields)
}

// Error logs an error level message
// Error mencatat pesan level error
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
    l.logWithFields(ERROR, msg, fields)
}

// Fatal logs a fatal level message and exits
// Fatal mencatat pesan level fatal dan keluar
func (l *Logger) Fatal(msg string, fields ...map[string]interface{}) {
    l.logWithFields(FATAL, msg, fields)
}

// Panic logs a panic level message and panics
// Panic mencatat pesan level panic dan panic
func (l *Logger) Panic(msg string, fields ...map[string]interface{}) {
    l.logWithFields(PANIC, msg, fields)
}

// logWithFields logs a message with fields
// logWithFields mencatat pesan dengan field
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

// ===============================
// CONTEXT-AWARE LOGGING METHODS - OPTIMIZED VERSION
// ===============================
// TraceContext logs a trace level message with context
// TraceContext mencatat pesan level trace dengan konteks
func (l *Logger) TraceContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(TRACE, ctx, msg, fields)
}

// DebugContext logs a debug level message with context
// DebugContext mencatat pesan level debug dengan konteks
func (l *Logger) DebugContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(DEBUG, ctx, msg, fields)
}

// InfoContext logs an info level message with context
// InfoContext mencatat pesan level info dengan konteks
func (l *Logger) InfoContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(INFO, ctx, msg, fields)
}

// NoticeContext logs a notice level message with context
// NoticeContext mencatat pesan level notice dengan konteks
func (l *Logger) NoticeContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(NOTICE, ctx, msg, fields)
}

// WarnContext logs a warning level message with context
// WarnContext mencatat pesan level warning dengan konteks
func (l *Logger) WarnContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(WARN, ctx, msg, fields)
}

// ErrorContext logs an error level message with context
// ErrorContext mencatat pesan level error dengan konteks
func (l *Logger) ErrorContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(ERROR, ctx, msg, fields)
}

// FatalContext logs a fatal level message with context and exits
// FatalContext mencatat pesan level fatal dengan konteks dan keluar
func (l *Logger) FatalContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(FATAL, ctx, msg, fields)
}

// PanicContext logs a panic level message with context and panics
// PanicContext mencatat pesan level panic dengan konteks dan panic
func (l *Logger) PanicContext(ctx context.Context, msg string, fields ...map[string]interface{}) {
    l.logWithFieldsContext(PANIC, ctx, msg, fields)
}

// logWithFieldsContext logs a message with fields and context
// logWithFieldsContext mencatat pesan dengan field dan konteks
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

// ===============================
// ERROR LOGGING METHODS - OPTIMIZED VERSION
// ===============================
// ErrorErr logs an error with error information
// ErrorErr mencatat kesalahan dengan informasi kesalahan
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

// ErrorErrContext logs an error with error information and context
// ErrorErrContext mencatat kesalahan dengan informasi kesalahan dan konteks
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

// ===============================
// LOGGER MANAGEMENT METHODS - OPTIMIZED VERSION
// ===============================
// WithField returns a new logger with an additional field
// WithField mengembalikan logger baru dengan field tambahan
func (l *Logger) WithField(key string, value interface{}) *Logger {
    newLogger := NewLogger(l.config)
    newLogger.fields = make(map[string]interface{}, len(l.fields)+1)
    for k, v := range l.fields {
        newLogger.fields[k] = v
    }
    newLogger.fields[key] = value
    return newLogger
}

// WithFields returns a new logger with additional fields
// WithFields mengembalikan logger baru dengan field tambahan
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

// WithContextExtractor sets a custom context extractor
// WithContextExtractor mengekstrak konteks kustom
func (l *Logger) WithContextExtractor(extractor func(context.Context) map[string]string) *Logger {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.contextExtractor = extractor
    return l
}

// AddHook adds a hook to the logger
// AddHook menambahkan hook ke logger
func (l *Logger) AddHook(hook func(*LogEntry)) {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.hooks = append(l.hooks, hook)
}

// SetOutput sets the output writer
// SetOutput mengatur writer output
func (l *Logger) SetOutput(w io.Writer) {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.out = w
}

// SetErrorOutput sets the error output writer
// SetErrorOutput mengatur writer output kesalahan
func (l *Logger) SetErrorOutput(w io.Writer) {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.errOut = w
}

// SetFormatter sets the formatter
// SetFormatter mengatur formatter
func (l *Logger) SetFormatter(formatter Formatter) {
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    l.formatter = formatter
}

// Close closes the logger
// Close menutup logger
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

// ===============================
// FORMATTED LOGGING METHODS - OPTIMIZED VERSION
// ===============================
// Tracef logs a formatted trace level message
// Tracef mencatat pesan level trace yang diformat
func (l *Logger) Tracef(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(TRACE, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(TRACE, fmt.Sprintf(format, args...), nil, nil)
    }
}

// Debugf logs a formatted debug level message
// Debugf mencatat pesan level debug yang diformat
func (l *Logger) Debugf(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(DEBUG, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(DEBUG, fmt.Sprintf(format, args...), nil, nil)
    }
}

// Infof logs a formatted info level message
// Infof mencatat pesan level info yang diformat
func (l *Logger) Infof(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(INFO, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(INFO, fmt.Sprintf(format, args...), nil, nil)
    }
}

// Noticef logs a formatted notice level message
// Noticef mencatat pesan level notice yang diformat
func (l *Logger) Noticef(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(NOTICE, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(NOTICE, fmt.Sprintf(format, args...), nil, nil)
    }
}

// Warnf logs a formatted warning level message
// Warnf mencatat pesan level warning yang diformat
func (l *Logger) Warnf(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(WARN, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(WARN, fmt.Sprintf(format, args...), nil, nil)
    }
}

// Errorf logs a formatted error level message
// Errorf mencatat pesan level error yang diformat
func (l *Logger) Errorf(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(ERROR, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(ERROR, fmt.Sprintf(format, args...), nil, nil)
    }
}

// Fatalf logs a formatted fatal level message and exits
// Fatalf mencatat pesan level fatal yang diformat dan keluar
func (l *Logger) Fatalf(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(FATAL, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(FATAL, fmt.Sprintf(format, args...), nil, nil)
    }
}

// Panicf logs a formatted panic level message and panics
// Panicf mencatat pesan level panic yang diformat dan panic
func (l *Logger) Panicf(format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(PANIC, fmt.Sprintf(format, args...), nil, nil)
    } else {
        l.log(PANIC, fmt.Sprintf(format, args...), nil, nil)
    }
}

// ===============================
// FIELDS-BASED LOGGING METHODS - OPTIMIZED VERSION
// ===============================
// TraceWithFields logs a trace level message with fields
// TraceWithFields mencatat pesan level trace dengan field
func (l *Logger) TraceWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(TRACE, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(TRACE, fmt.Sprintf(format, args...), fields, nil)
    }
}

// DebugWithFields logs a debug level message with fields
// DebugWithFields mencatat pesan level debug dengan field
func (l *Logger) DebugWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(DEBUG, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(DEBUG, fmt.Sprintf(format, args...), fields, nil)
    }
}

// InfoWithFields logs an info level message with fields
// InfoWithFields mencatat pesan level info dengan field
func (l *Logger) InfoWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(INFO, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(INFO, fmt.Sprintf(format, args...), fields, nil)
    }
}

// NoticeWithFields logs a notice level message with fields
// NoticeWithFields mencatat pesan level notice dengan field
func (l *Logger) NoticeWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(NOTICE, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(NOTICE, fmt.Sprintf(format, args...), fields, nil)
    }
}

// WarnWithFields logs a warning level message with fields
// WarnWithFields mencatat pesan level warning dengan field
func (l *Logger) WarnWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(WARN, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(WARN, fmt.Sprintf(format, args...), fields, nil)
    }
}

// ErrorWithFields logs an error level message with fields
// ErrorWithFields mencatat pesan level error dengan field
func (l *Logger) ErrorWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(ERROR, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(ERROR, fmt.Sprintf(format, args...), fields, nil)
    }
}

// FatalWithFields logs a fatal level message with fields and exits
// FatalWithFields mencatat pesan level fatal dengan field dan keluar
func (l *Logger) FatalWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(FATAL, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(FATAL, fmt.Sprintf(format, args...), fields, nil)
    }
}

// PanicWithFields logs a panic level message with fields and panics
// PanicWithFields mencatat pesan level panic dengan field dan panic
func (l *Logger) PanicWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    if l.asyncLogger != nil {
        l.asyncLogger.log(PANIC, fmt.Sprintf(format, args...), fields, nil)
    } else {
        l.log(PANIC, fmt.Sprintf(format, args...), fields, nil)
    }
}

// ===============================
// ENHANCED LOGGING METHODS - OPTIMIZED VERSION
// ===============================
// log is the core logging method with optimizations
// log adalah metode logging inti dengan optimasi
func (l *Logger) log(level Level, msg string, fields map[string]interface{}, ctx context.Context) {
    // Optimization: Fast path check
    // Optimasi: Pemeriksaan jalur cepat
    if level < l.config.Level {
        return
    }
    
    // Optimization: Sampling check
    // Optimasi: Pemeriksaan sampling
    if l.sampler != nil && !l.sampler.shouldLog() {
        return
    }
    
    // Optimization: Message size check
    // Optimasi: Pemeriksaan ukuran pesan
    if l.config.MaxMessageSize > 0 && len(msg) > l.config.MaxMessageSize {
        msg = msg[:l.config.MaxMessageSize] + "... [truncated]"
    }
    
    start := time.Now()
    
    // Optimization: Use object pool to get LogEntry
    // Optimasi: Menggunakan pool objek untuk mendapatkan LogEntry
    entry := getEntryFromPool()
    defer putEntryToPool(entry)
    
    // Pre-allocate field capacity (maps don't have cap, so we just make a new map if needed)
    // Pra-alokasi kapasitas field (map tidak memiliki cap, jadi kita buat map baru jika diperlukan)
    if len(l.fields)+len(fields) > len(entry.Fields) {
        entry.Fields = make(map[string]interface{}, len(l.fields)+len(fields))
    } else {
        // Reset map but keep capacity
        // Reset map tetapi pertahankan kapasitas
        for k := range entry.Fields {
            delete(entry.Fields, k)
        }
    }
    
    // Merge fields
    // Menggabungkan field
    for k, v := range l.fields {
        entry.Fields[k] = v
    }
    for k, v := range fields {
        entry.Fields[k] = v
    }
    
    // Fill basic fields
    // Mengisi field dasar
    entry.Timestamp = time.Now()
    entry.Level = level
    entry.LevelName = level.String()
    entry.Message = msg
    entry.PID = os.Getpid()
    entry.Fields = entry.Fields
    entry.Hostname = l.config.Hostname
    entry.Application = l.config.Application
    entry.Version = l.config.Version
    entry.Environment = l.config.Environment
    
    // Optimization: Context extraction
    // Optimasi: Ekstraksi konteks
    if ctx != nil {
        contextValues := make(map[string]string, 5) // Pre-allocate capacity
        
        if l.contextExtractor != nil {
            contextValues = l.contextExtractor(ctx)
        } else {
            contextValues = ExtractFromContext(ctx)
        }
        
        if traceID, ok := contextValues["trace_id"]; ok {
            entry.TraceID = traceID
        }
        if spanID, ok := contextValues["span_id"]; ok {
            entry.SpanID = spanID
        }
        if userID, ok := contextValues["user_id"]; ok {
            entry.UserID = userID
        }
        if sessionID, ok := contextValues["session_id"]; ok {
            entry.SessionID = sessionID
        }
        if requestID, ok := contextValues["request_id"]; ok {
            entry.RequestID = requestID
        }
    }
    
    // Optimization: Caller information retrieval
    // Optimasi: Pengambilan informasi pemanggil
    if l.config.ShowCaller {
        if pc, file, line, ok := runtime.Caller(l.config.CallerDepth); ok {
            function := runtime.FuncForPC(pc).Name()
            pkg := filepath.Dir(function)
            
            entry.Caller = &CallerInfo{
                File:     filepath.Base(file),
                Line:     line,
                Function: filepath.Base(function),
                Package:  pkg,
            }
        }
    }
    
    // Optimization: Goroutine ID retrieval
    // Optimasi: Pengambilan ID goroutine
    if l.config.ShowGoroutine {
        entry.GoroutineID = getGoroutineID()
    }
    
    // Optimization: Stack trace retrieval
    // Optimasi: Pengambilan stack trace
    if l.config.EnableStackTrace && (level >= ERROR) {
        entry.StackTrace = getStackTrace(l.config.StackTraceDepth)
    }
    
    // Execute hooks
    // Menjalankan hook
    for _, hook := range l.hooks {
        hook(entry)
    }
    
    // Optimization: Conditional locking
    // Optimasi: Penguncian bersyarat
    if !l.config.DisableLocking {
        l.mu.Lock()
        defer l.mu.Unlock()
    }
    
    // Optimization: Formatting
    // Optimasi: Pemformatan
    formatted, err := l.formatter.Format(entry)
    if err != nil {
        if l.errorHandler != nil {
            l.errorHandler(err)
        } else {
            fmt.Fprintf(l.errOut, "Logger error: %v\n", err)
        }
        return
    }
    
    // Optimization: Writing
    // Optimasi: Penulisan
    bytesWritten, err := l.out.Write(formatted)
    if err != nil {
        if l.errorHandler != nil {
            l.errorHandler(err)
        }
    }
    
    // Update statistics
    // Memperbarui statistik
    l.stats.Increment(level, bytesWritten)
    
    // Optimization: Metrics collection
    // Optimasi: Pengumpulan metrik
    if l.metrics != nil {
        tags := map[string]string{
            "level": level.String(),
            "application": l.config.Application,
            "environment": l.config.Environment,
        }
        l.metrics.IncrementCounter(level, tags)
    }
    
    // Handle fatal and panic levels
    // Menangani level fatal dan panic
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
    
    // Optimization: Slow log detection
    // Optimasi: Deteksi log lambat
    if time.Since(start) > 100*time.Millisecond {
        slowFields := map[string]interface{}{
            "logging_duration": time.Since(start).String(),
            "message_length":   len(msg),
            "fields_count":     len(entry.Fields),
        }
        l.log(WARN, "Slow logging operation", slowFields, nil)
    }
}

// ===============================
// BATCH LOGGING - OPTIMIZED VERSION
// ===============================
// LogBatch logs multiple entries in a batch for better performance
// LogBatch mencatat beberapa entri dalam batch untuk performa lebih baik
func (l *Logger) LogBatch(entries []*LogEntry) error {
    if len(entries) == 0 {
        return nil
    }
    
    // Optimization: Pre-calculate total size
    // Optimasi: Pra-hitung ukuran total
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
    
    // Optimization: Combine all formatted entries
    // Optimasi: Menggabungkan semua entri yang diformat
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

// ===============================
// AUDIT LOGGING - OPTIMIZED VERSION
// ===============================
// Audit logs an audit event
// Audit mencatat peristiwa audit
func (l *Logger) Audit(eventType, action, resource string, userID interface{}, success bool, details map[string]interface{}) {
    // Optimization: Pre-allocate fields
    // Optimasi: Pra-alokasi field
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

// ===============================
// PERFORMANCE MONITORING - OPTIMIZED VERSION
// ===============================
// TimeOperation measures and logs operation duration
// TimeOperation mengukur dan mencatat durasi operasi
func (l *Logger) TimeOperation(operationName string, fields map[string]interface{}, operation func()) {
    start := time.Now()
    
    defer func() {
        duration := time.Since(start)
        
        // Optimization: Pre-allocate fields
        // Optimasi: Pra-alokasi field
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
        
        // Optimization: Metrics collection
        // Optimasi: Pengumpulan metrik
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
// COMPREHENSIVE ROTATING FILE WRITER - OPTIMIZED VERSION
// ===============================
// RotatingFileWriter provides log rotation with compression
// RotatingFileWriter menyediakan rotasi log dengan kompresi
type RotatingFileWriter struct {
    filename       string         // Current filename
    maxSize        int64          // Maximum file size
    maxAge         time.Duration  // Maximum file age
    maxBackups     int            // Maximum backup files
    localTime      bool           // Use local time
    compress       bool           // Compress rotated files
    rotationTime   time.Duration  // Time-based rotation
    currentSize    int64          // Current file size
    file           *os.File       // Current file handle
    mu             sync.Mutex     // Mutex for thread safety
    lastRotation   time.Time      // Time of last rotation
    pattern        string         // Filename pattern
    compressedExt  string         // Extension for compressed files
    
    // Optimization: Buffer pool
    // Optimasi: Pool buffer
    bufferPool     sync.Pool      // Pool for reusing buffers
}

// NewRotatingFileWriter creates a new rotating file writer
// NewRotatingFileWriter membuat writer file rotasi baru
func NewRotatingFileWriter(filename string, config *RotationConfig) (*RotatingFileWriter, error) {
    r := &RotatingFileWriter{
        filename:      filename,
        compress:      config.Compress,
        compressedExt: ".gz",
        bufferPool: sync.Pool{
            New: func() interface{} {
                return make([]byte, 0, 64*1024) // 64KB buffer
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
    // Inisialisasi file
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
// Write menulis data ke file rotasi
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
// needsRotation menentukan apakah rotasi diperlukan
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
// rotate melakukan rotasi file yang sebenarnya
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
// compressFile mengompres file menggunakan gzip
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
    
    // Optimization: Use buffer pool
    // Optimasi: Menggunakan pool buffer
    buf := r.bufferPool.Get().([]byte)
    defer r.bufferPool.Put(buf[:0])
    
    _, err = io.CopyBuffer(gzWriter, srcFile, buf)
    if err != nil {
        return err
    }
    
    return os.Remove(srcFilename)
}

// cleanupBackups removes old backup files
// cleanupBackups menghapus file cadangan lama
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
// Close menutup writer file rotasi
func (r *RotatingFileWriter) Close() error {
    r.mu.Lock()
    defer r.mu.Unlock()
    if r.file != nil {
        return r.file.Close()
    }
    return nil
}

// ===============================
// UTILITY FUNCTIONS - OPTIMIZED VERSION
// ===============================

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

// contains checks if a slice contains a string - Optimized version
// contains memeriksa apakah slice mengandung string - Versi optimal
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

// shortID shortens an ID for display
// shortID mempersingkat ID untuk ditampilkan
func shortID(id string) string {
    if len(id) <= 8 {
        return id
    }
    return id[:8]
}

// getGoroutineID gets the current goroutine ID
// getGoroutineID mendapatkan ID goroutine saat ini
func getGoroutineID() string {
    var buf [64]byte
    n := runtime.Stack(buf[:], false)
    idField := strings.TrimPrefix(string(buf[:n]), "goroutine ")
    idField = idField[:strings.Index(idField, " ")]
    return idField
}

// getStackTrace gets the current stack trace
// getStackTrace mendapatkan stack trace saat ini
func getStackTrace(depth int) string {
    buf := make([]byte, 1024)
    for {
        n := runtime.Stack(buf, false)
        if n < len(buf) {
            return string(buf[:n])
        }
        buf = make([]byte, 2*len(buf))
    }
}

// formatValue formats a value for display - Optimized version
// formatValue memformat nilai untuk ditampilkan - Versi optimal
func formatValue(v interface{}, maxWidth int) string {
    if v == nil {
        return "null"
    }
    
    switch val := v.(type) {
    case string:
        if maxWidth > 0 && len(val) > maxWidth {
            return val[:maxWidth] + "..."
        }
        return val
    case error:
        s := val.Error()
        if maxWidth > 0 && len(s) > maxWidth {
            return s[:maxWidth] + "..."
        }
        return s
    case []byte:
        s := string(val)
        if maxWidth > 0 && len(s) > maxWidth {
            return s[:maxWidth] + "..."
        }
        return s
    default:
        s := fmt.Sprintf("%v", val)
        if maxWidth > 0 && len(s) > maxWidth {
            return s[:maxWidth] + "..."
        }
        return s
    }
}
