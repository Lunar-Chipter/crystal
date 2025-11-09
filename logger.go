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
    "unicode/utf8"
    "context"
    "net/http"
    "encoding/base64"
    "crypto/rand"
    "compress/gzip"
    "encoding/csv"
    "math"
    "sort"
    "bytes"
    "errors"
    "io/ioutil"
)

const (
    DEFAULT_TIMESTAMP_FORMAT = "2006-01-02 15:04:05.000"
    DEFAULT_CALLER_DEPTH     = 3
    DEFAULT_BUFFER_SIZE      = 1000
    DEFAULT_FLUSH_INTERVAL   = 5 * time.Second
)

// ===============================
// LEVEL DEFINITION
// ===============================
type Level int

const (
    TRACE Level = iota
    DEBUG
    INFO
    NOTICE
    WARN
    ERROR
    FATAL
    PANIC
)

var (
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
func (l Level) String() string {
    if l >= TRACE && l <= PANIC {
        return levelStrings[l]
    }
    return "UNKNOWN"
}

// ParseLevel parses level from string
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
// LOG ENTRY STRUCT
// ===============================
type LogEntry struct {
    Timestamp     time.Time              `json:"timestamp"`
    Level         Level                  `json:"level"`
    LevelName     string                 `json:"level_name"`
    Message       string                 `json:"message"`
    Caller        *CallerInfo            `json:"caller,omitempty"`
    Fields        map[string]interface{} `json:"fields,omitempty"`
    PID           int                    `json:"pid"`
    GoroutineID   string                 `json:"goroutine_id,omitempty"`
    TraceID       string                 `json:"trace_id,omitempty"`
    SpanID        string                 `json:"span_id,omitempty"`
    UserID        string                 `json:"user_id,omitempty"`
    SessionID     string                 `json:"session_id,omitempty"`
    RequestID     string                 `json:"request_id,omitempty"`
    Duration      time.Duration          `json:"duration,omitempty"`
    Error         error                  `json:"error,omitempty"`
    StackTrace    string                 `json:"stack_trace,omitempty"`
    Hostname      string                 `json:"hostname,omitempty"`
    Application   string                 `json:"application,omitempty"`
    Version       string                 `json:"version,omitempty"`
    Environment   string                 `json:"environment,omitempty"`
    CustomMetrics map[string]float64     `json:"custom_metrics,omitempty"`
    Tags          []string               `json:"tags,omitempty"`
}

type CallerInfo struct {
    File     string `json:"file"`
    Line     int    `json:"line"`
    Function string `json:"function"`
    Package  string `json:"package"`
}

// ===============================
// CONTEXT SUPPORT
// ===============================
type contextKey string

const (
    TraceIDKey     contextKey = "trace_id"
    SpanIDKey      contextKey = "span_id"
    UserIDKey      contextKey = "user_id"
    SessionIDKey   contextKey = "session_id"
    RequestIDKey   contextKey = "request_id"
    ClientIPKey    contextKey = "client_ip"
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

// ExtractFromContext extracts all context values
func ExtractFromContext(ctx context.Context) map[string]string {
    result := make(map[string]string)
    
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
// ADVANCED FORMATTERS
// ===============================
type Formatter interface {
    Format(entry *LogEntry) (string, error)
}

// TextFormatter dengan format detail dan berwarna
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
    IndentFields          bool
    MaxFieldWidth         int
    EnableStackTrace      bool
    StackTraceDepth       int
    EnableDuration        bool
    CustomFieldOrder      []string
    EnableColorsByLevel   bool
    FieldTransformers     map[string]func(interface{}) string
    SensitiveFields       []string
    MaskSensitiveData     bool
    MaskString            string
}

func (f *TextFormatter) Format(entry *LogEntry) (string, error) {
    var buf strings.Builder
    
    // Timestamp
    if f.ShowTimestamp {
        timestamp := entry.Timestamp.Format(f.TimestampFormat)
        if f.EnableColors {
            buf.WriteString("\033[38;5;244m") // Gray
        }
        buf.WriteString("[")
        buf.WriteString(timestamp)
        buf.WriteString("]")
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteString(" ")
    }
    
    // Level dengan background dan padding
    levelStr := levelStrings[entry.Level]
    if f.EnableColors {
        buf.WriteString(levelBackgrounds[entry.Level])
        buf.WriteString(levelColors[entry.Level])
        buf.WriteString(" ")
        buf.WriteString(levelStr)
        buf.WriteString(" ")
        buf.WriteString("\033[0m")
    } else {
        buf.WriteString("[")
        buf.WriteString(levelStr)
        buf.WriteString("]")
    }
    buf.WriteString(" ")
    
    // Hostname
    if f.ShowHostname && entry.Hostname != "" {
        if f.EnableColors {
            buf.WriteString("\033[38;5;245m")
        }
        buf.WriteString(entry.Hostname)
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteString(" ")
    }
    
    // Application
    if f.ShowApplication && entry.Application != "" {
        if f.EnableColors {
            buf.WriteString("\033[38;5;245m")
        }
        buf.WriteString(entry.Application)
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteString(" ")
    }
    
    // PID
    if f.ShowPID {
        if f.EnableColors {
            buf.WriteString("\033[38;5;245m")
        }
        buf.WriteString("PID:")
        buf.WriteString(strconv.Itoa(entry.PID))
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteString(" ")
    }
    
    // Goroutine ID
    if f.ShowGoroutine && entry.GoroutineID != "" {
        if f.EnableColors {
            buf.WriteString("\033[38;5;245m")
        }
        buf.WriteString("GID:")
        buf.WriteString(entry.GoroutineID)
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteString(" ")
    }
    
    // Trace Information
    if f.ShowTraceInfo {
        if entry.TraceID != "" {
            if f.EnableColors {
                buf.WriteString("\033[38;5;141m") // Purple
            }
            buf.WriteString("TRACE:")
            buf.WriteString(shortID(entry.TraceID))
            if f.EnableColors {
                buf.WriteString("\033[0m")
            }
            buf.WriteString(" ")
        }
        
        if entry.SpanID != "" {
            if f.EnableColors {
                buf.WriteString("\033[38;5;141m") // Purple
            }
            buf.WriteString("SPAN:")
            buf.WriteString(shortID(entry.SpanID))
            if f.EnableColors {
                buf.WriteString("\033[0m")
            }
            buf.WriteString(" ")
        }
        
        if entry.RequestID != "" {
            if f.EnableColors {
                buf.WriteString("\033[38;5;141m") // Purple
            }
            buf.WriteString("REQ:")
            buf.WriteString(shortID(entry.RequestID))
            if f.EnableColors {
                buf.WriteString("\033[0m")
            }
            buf.WriteString(" ")
        }
    }
    
    // Caller Information
    if f.ShowCaller && entry.Caller != nil {
        if f.EnableColors {
            buf.WriteString("\033[38;5;246m")
        }
        buf.WriteString(entry.Caller.File)
        buf.WriteString(":")
        buf.WriteString(strconv.Itoa(entry.Caller.Line))
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteString(" ")
    }
    
    // Duration
    if f.EnableDuration && entry.Duration > 0 {
        if f.EnableColors {
            buf.WriteString("\033[38;5;155m") // Light green
        }
        buf.WriteString("(")
        buf.WriteString(entry.Duration.String())
        buf.WriteString(")")
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
        buf.WriteString(" ")
    }
    
    // Message dengan warna berdasarkan level
    if f.EnableColors && f.EnableColorsByLevel {
        buf.WriteString(levelColors[entry.Level])
        if entry.Level >= ERROR {
            buf.WriteString("\033[1m") // Bold for important messages
        }
    }
    buf.WriteString(entry.Message)
    if f.EnableColors && f.EnableColorsByLevel {
        buf.WriteString("\033[0m")
    }
    
    // Error
    if entry.Error != nil {
        buf.WriteString(" ")
        if f.EnableColors {
            buf.WriteString("\033[38;5;196m") // Bright red for errors
        }
        buf.WriteString("error=\"")
        buf.WriteString(entry.Error.Error())
        buf.WriteString("\"")
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
    }
    
    // Fields
    if len(entry.Fields) > 0 {
        buf.WriteString(" ")
        f.formatFields(&buf, entry.Fields)
    }
    
    // Tags
    if len(entry.Tags) > 0 {
        buf.WriteString(" ")
        f.formatTags(&buf, entry.Tags)
    }
    
    // Metrics
    if len(entry.CustomMetrics) > 0 {
        buf.WriteString(" ")
        f.formatMetrics(&buf, entry.CustomMetrics)
    }
    
    // Stack Trace
    if f.EnableStackTrace && entry.StackTrace != "" {
        buf.WriteString("\n")
        if f.EnableColors {
            buf.WriteString("\033[38;5;240m") // Dark gray for stack trace
        }
        buf.WriteString(entry.StackTrace)
        if f.EnableColors {
            buf.WriteString("\033[0m")
        }
    }
    
    buf.WriteString("\n")
    return buf.String(), nil
}

func (f *TextFormatter) formatFields(buf *strings.Builder, fields map[string]interface{}) {
    if f.EnableColors {
        buf.WriteString("\033[38;5;243m")
    }
    buf.WriteString("{")
    
    // Use custom field order if specified
    keys := make([]string, 0, len(fields))
    if len(f.CustomFieldOrder) > 0 {
        // Add ordered fields first
        for _, key := range f.CustomFieldOrder {
            if _, exists := fields[key]; exists {
                keys = append(keys, key)
            }
        }
        // Add remaining fields
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
            buf.WriteString(" ")
        }
        
        if f.EnableColors {
            buf.WriteString("\033[38;5;228m") // Light yellow for keys
        }
        buf.WriteString(k)
        buf.WriteString("=")
        if f.EnableColors {
            buf.WriteString("\033[38;5;159m") // Light cyan for values
        }
        
        // Apply field transformer if exists
        if transformer, exists := f.FieldTransformers[k]; exists {
            buf.WriteString(transformer(v))
        } else {
            // Mask sensitive data
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
    buf.WriteString("}")
    if f.EnableColors {
        buf.WriteString("\033[0m")
    }
}

func (f *TextFormatter) formatTags(buf *strings.Builder, tags []string) {
    if f.EnableColors {
        buf.WriteString("\033[38;5;135m") // Purple for tags
    }
    buf.WriteString("[")
    for i, tag := range tags {
        if i > 0 {
            buf.WriteString(",")
        }
        buf.WriteString(tag)
    }
    buf.WriteString("]")
    if f.EnableColors {
        buf.WriteString("\033[0m")
    }
}

func (f *TextFormatter) formatMetrics(buf *strings.Builder, metrics map[string]float64) {
    if f.EnableColors {
        buf.WriteString("\033[38;5;85m") // Green for metrics
    }
    buf.WriteString("<")
    first := true
    for k, v := range metrics {
        if !first {
            buf.WriteString(" ")
        }
        first = false
        buf.WriteString(k)
        buf.WriteString("=")
        buf.WriteString(strconv.FormatFloat(v, 'f', 2, 64))
    }
    buf.WriteString(">")
    if f.EnableColors {
        buf.WriteString("\033[0m")
    }
}

// JSONFormatter untuk output structured logging
type JSONFormatter struct {
    PrettyPrint          bool
    TimestampFormat      string
    ShowCaller           bool
    ShowGoroutine        bool
    ShowPID              bool
    ShowTraceInfo        bool
    EnableStackTrace     bool
    EnableDuration       bool
    FieldKeyMap          map[string]string
    DisableHTMLEscape    bool
    SensitiveFields      []string
    MaskSensitiveData    bool
    MaskString           string
    FieldTransformers    map[string]func(interface{}) interface{}
}

func (f *JSONFormatter) Format(entry *LogEntry) (string, error) {
    // Buat copy untuk menghindari modifikasi original
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
    
    // Format timestamp jika diperlukan
    if f.TimestampFormat != "" {
        outputEntry.Timestamp = entry.Timestamp
    }
    
    var data []byte
    var err error
    
    if f.PrettyPrint {
        data, err = json.MarshalIndent(outputEntry, "", "  ")
    } else {
        var buf bytes.Buffer
        encoder := json.NewEncoder(&buf)
        if f.DisableHTMLEscape {
            encoder.SetEscapeHTML(false)
        }
        err = encoder.Encode(outputEntry)
        data = buf.Bytes()
    }
    
    if err != nil {
        return "", err
    }
    
    return string(data), nil
}

func (f *JSONFormatter) processFields(fields map[string]interface{}) map[string]interface{} {
    processed := make(map[string]interface{})
    
    for k, v := range fields {
        // Apply field mapping
        key := k
        if mappedKey, exists := f.FieldKeyMap[k]; exists {
            key = mappedKey
        }
        
        // Apply transformer
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

// CSVFormatter untuk output dalam format CSV
type CSVFormatter struct {
    IncludeHeader    bool
    FieldOrder       []string
    TimestampFormat  string
}

func (f *CSVFormatter) Format(entry *LogEntry) (string, error) {
    var buf strings.Builder
    writer := csv.NewWriter(&buf)
    
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
        return "", err
    }
    
    writer.Flush()
    return buf.String(), nil
}

// ===============================
// LOGGER CONFIGURATION
// ===============================
type LoggerConfig struct {
    Level              Level
    EnableColors       bool
    Output             io.Writer
    ErrorOutput        io.Writer
    Formatter          Formatter
    ShowCaller         bool
    CallerDepth        int
    ShowGoroutine      bool
    ShowPID            bool
    ShowTraceInfo      bool
    ShowHostname       bool
    ShowApplication    bool
    TimestampFormat    string
    ExitFunc           func(int)
    EnableStackTrace   bool
    StackTraceDepth    int
    EnableSampling     bool
    SamplingRate       int
    BufferSize         int
    FlushInterval      time.Duration
    EnableRotation     bool
    RotationConfig     *RotationConfig
    ContextExtractor   func(context.Context) map[string]string
    Hostname           string
    Application        string
    Version           string
    Environment        string
    MaxFieldSize       int
    EnableMetrics      bool
    MetricsCollector   MetricsCollector
    ErrorHandler       func(error)
    OnFatal            func(*LogEntry)
    OnPanic            func(*LogEntry)
}

type RotationConfig struct {
    MaxSize         int64
    MaxAge          time.Duration
    MaxBackups      int
    LocalTime       bool
    Compress        bool
    RotationTime    time.Duration
    FilenamePattern string
}

// ===============================
// METRICS COLLECTOR
// ===============================
type MetricsCollector interface {
    IncrementCounter(level Level, tags map[string]string)
    RecordHistogram(metric string, value float64, tags map[string]string)
    RecordGauge(metric string, value float64, tags map[string]string)
}

type DefaultMetricsCollector struct {
    counters   map[string]int64
    histograms map[string][]float64
    gauges     map[string]float64
    mu         sync.RWMutex
}

func NewDefaultMetricsCollector() *DefaultMetricsCollector {
    return &DefaultMetricsCollector{
        counters:   make(map[string]int64),
        histograms: make(map[string][]float64),
        gauges:     make(map[string]float64),
    }
}

func (d *DefaultMetricsCollector) IncrementCounter(level Level, tags map[string]string) {
    key := fmt.Sprintf("log.%s", strings.ToLower(level.String()))
    d.mu.Lock()
    defer d.mu.Unlock()
    d.counters[key]++
}

func (d *DefaultMetricsCollector) RecordHistogram(metric string, value float64, tags map[string]string) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.histograms[metric] = append(d.histograms[metric], value)
}

func (d *DefaultMetricsCollector) RecordGauge(metric string, value float64, tags map[string]string) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.gauges[metric] = value
}

func (d *DefaultMetricsCollector) GetCounter(metric string) int64 {
    d.mu.RLock()
    defer d.mu.RUnlock()
    return d.counters[metric]
}

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
// ADVANCED BUFFERED WRITER
// ===============================
type BufferedWriter struct {
    writer         io.Writer
    buffer         chan []byte
    bufferSize     int
    flushInterval  time.Duration
    done           chan struct{}
    wg             sync.WaitGroup
    mu             sync.Mutex
    droppedLogs    int64
    totalLogs      int64
    lastFlush      time.Time
}

func NewBufferedWriter(writer io.Writer, bufferSize int, flushInterval time.Duration) *BufferedWriter {
    bw := &BufferedWriter{
        writer:        writer,
        buffer:        make(chan []byte, bufferSize),
        bufferSize:    bufferSize,
        flushInterval: flushInterval,
        done:          make(chan struct{}),
        lastFlush:     time.Now(),
    }
    
    bw.wg.Add(1)
    go bw.flushWorker()
    
    return bw
}

func (bw *BufferedWriter) Write(p []byte) (n int, err error) {
    atomic.AddInt64(&bw.totalLogs, 1)
    
    // Make a copy to avoid data race
    data := make([]byte, len(p))
    copy(data, p)
    
    select {
    case bw.buffer <- data:
        return len(p), nil
    default:
        // Buffer full, drop the log and increment counter
        atomic.AddInt64(&bw.droppedLogs, 1)
        // Try direct write as fallback
        bw.mu.Lock()
        defer bw.mu.Unlock()
        return bw.writer.Write(p)
    }
}

func (bw *BufferedWriter) flushWorker() {
    defer bw.wg.Done()
    
    ticker := time.NewTicker(bw.flushInterval)
    defer ticker.Stop()
    
    batch := make([][]byte, 0, bw.bufferSize)
    
    for {
        select {
        case <-bw.done:
            bw.flushBatch(batch)
            return
        case <-ticker.C:
            bw.flushBatch(batch)
            batch = batch[:0] // Reset batch
            bw.lastFlush = time.Now()
        case data := <-bw.buffer:
            batch = append(batch, data)
            if len(batch) >= bw.bufferSize {
                bw.flushBatch(batch)
                batch = batch[:0]
                bw.lastFlush = time.Now()
            }
        }
    }
}

func (bw *BufferedWriter) flushBatch(batch [][]byte) {
    if len(batch) == 0 {
        return
    }
    
    bw.mu.Lock()
    defer bw.mu.Unlock()
    
    for _, data := range batch {
        bw.writer.Write(data)
    }
}

func (bw *BufferedWriter) Stats() map[string]interface{} {
    return map[string]interface{}{
        "buffer_size":    bw.bufferSize,
        "current_queue":  len(bw.buffer),
        "dropped_logs":   atomic.LoadInt64(&bw.droppedLogs),
        "total_logs":     atomic.LoadInt64(&bw.totalLogs),
        "last_flush":     bw.lastFlush,
    }
}

func (bw *BufferedWriter) Close() error {
    close(bw.done)
    bw.wg.Wait()
    
    // Flush remaining logs
    bw.flushBatch(nil) // This will flush all remaining in buffer
    
    if closer, ok := bw.writer.(io.Closer); ok {
        return closer.Close()
    }
    return nil
}

// ===============================
// SAMPLING LOGGER
// ===============================
type SamplingLogger struct {
    logger      *Logger
    rate        int
    counter     int64
    mu          sync.Mutex
}

func NewSamplingLogger(logger *Logger, rate int) *SamplingLogger {
    return &SamplingLogger{
        logger: logger,
        rate:   rate,
    }
}

func (sl *SamplingLogger) shouldLog() bool {
    if sl.rate <= 1 {
        return true
    }
    
    counter := atomic.AddInt64(&sl.counter, 1)
    return counter%int64(sl.rate) == 0
}

// ===============================
// MAIN LOGGER STRUCT
// ===============================
type Logger struct {
    config          LoggerConfig
    formatter       Formatter
    out             io.Writer
    errOut          io.Writer
    mu              sync.Mutex
    hooks           []func(*LogEntry)
    exitFunc        func(int)
    fields          map[string]interface{}
    sampler         *SamplingLogger
    buffer          *BufferedWriter
    rotation        *RotatingFileWriter
    contextExtractor func(context.Context) map[string]string
    metrics         MetricsCollector
    errorHandler    func(error)
    onFatal         func(*LogEntry)
    onPanic         func(*LogEntry)
    stats           *LoggerStats
}

type LoggerStats struct {
    LogCounts      map[Level]int64
    BytesWritten   int64
    StartTime      time.Time
    mu             sync.RWMutex
}

func NewLoggerStats() *LoggerStats {
    return &LoggerStats{
        LogCounts:    make(map[Level]int64),
        BytesWritten: 0,
        StartTime:    time.Now(),
    }
}

func (ls *LoggerStats) Increment(level Level, bytes int) {
    ls.mu.Lock()
    defer ls.mu.Unlock()
    ls.LogCounts[level]++
    ls.BytesWritten += int64(bytes)
}

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
// LOGGER METHODS
// ===============================
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

func (l *Logger) SetLevel(level Level) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.config.Level = level
}

func (l *Logger) GetStats() map[string]interface{} {
    return l.stats.GetStats()
}

func (l *Logger) Trace(msg string, fields ...map[string]interface{}) {
    l.logWithFields(TRACE, msg, fields)
}

func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
    l.logWithFields(DEBUG, msg, fields)
}

func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
    l.logWithFields(INFO, msg, fields)
}

func (l *Logger) Notice(msg string, fields ...map[string]interface{}) {
    l.logWithFields(NOTICE, msg, fields)
}

func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
    l.logWithFields(WARN, msg, fields)
}

func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
    l.logWithFields(ERROR, msg, fields)
}

func (l *Logger) Fatal(msg string, fields ...map[string]interface{}) {
    l.logWithFields(FATAL, msg, fields)
}

func (l *Logger) Panic(msg string, fields ...map[string]interface{}) {
    l.logWithFields(PANIC, msg, fields)
}

func (l *Logger) logWithFields(level Level, msg string, fields []map[string]interface{}) {
    var mergedFields map[string]interface{}
    if len(fields) > 0 {
        mergedFields = fields[0]
    }
    l.log(level, msg, mergedFields, nil)
}

// ===============================
// CONTEXT-AWARE LOGGING METHODS
// ===============================
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

func (l *Logger) logWithFieldsContext(level Level, ctx context.Context, msg string, fields []map[string]interface{}) {
    var mergedFields map[string]interface{}
    if len(fields) > 0 {
        mergedFields = fields[0]
    }
    l.log(level, msg, mergedFields, ctx)
}

// ===============================
// ERROR LOGGING METHODS
// ===============================
func (l *Logger) ErrorErr(err error, msg string, fields ...map[string]interface{}) {
    mergedFields := make(map[string]interface{})
    if len(fields) > 0 {
        mergedFields = fields[0]
    }
    mergedFields["error"] = err.Error()
    l.log(ERROR, msg, mergedFields, nil)
}

func (l *Logger) ErrorErrContext(ctx context.Context, err error, msg string, fields ...map[string]interface{}) {
    mergedFields := make(map[string]interface{})
    if len(fields) > 0 {
        mergedFields = fields[0]
    }
    mergedFields["error"] = err.Error()
    l.log(ERROR, msg, mergedFields, ctx)
}

// ===============================
// LOGGER MANAGEMENT METHODS
// ===============================
func (l *Logger) WithField(key string, value interface{}) *Logger {
    newLogger := NewLogger(l.config)
    newLogger.fields = make(map[string]interface{})
    for k, v := range l.fields {
        newLogger.fields[k] = v
    }
    newLogger.fields[key] = value
    return newLogger
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
    newLogger := NewLogger(l.config)
    newLogger.fields = make(map[string]interface{})
    for k, v := range l.fields {
        newLogger.fields[k] = v
    }
    for k, v := range fields {
        newLogger.fields[k] = v
    }
    return newLogger
}

func (l *Logger) WithContextExtractor(extractor func(context.Context) map[string]string) *Logger {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.contextExtractor = extractor
    return l
}

func (l *Logger) AddHook(hook func(*LogEntry)) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.hooks = append(l.hooks, hook)
}

func (l *Logger) SetOutput(w io.Writer) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.out = w
}

func (l *Logger) SetErrorOutput(w io.Writer) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.errOut = w
}

func (l *Logger) SetFormatter(formatter Formatter) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.formatter = formatter
}

func (l *Logger) Close() error {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    if l.buffer != nil {
        return l.buffer.Close()
    }
    
    if l.rotation != nil {
        return l.rotation.Close()
    }
    
    return nil
}

// ===============================
// FORMATTED LOGGING METHODS
// ===============================
func (l *Logger) Tracef(format string, args ...interface{}) {
    l.log(TRACE, fmt.Sprintf(format, args...), nil, nil)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
    l.log(DEBUG, fmt.Sprintf(format, args...), nil, nil)
}

func (l *Logger) Infof(format string, args ...interface{}) {
    l.log(INFO, fmt.Sprintf(format, args...), nil, nil)
}

func (l *Logger) Noticef(format string, args ...interface{}) {
    l.log(NOTICE, fmt.Sprintf(format, args...), nil, nil)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
    l.log(WARN, fmt.Sprintf(format, args...), nil, nil)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
    l.log(ERROR, fmt.Sprintf(format, args...), nil, nil)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
    l.log(FATAL, fmt.Sprintf(format, args...), nil, nil)
}

func (l *Logger) Panicf(format string, args ...interface{}) {
    l.log(PANIC, fmt.Sprintf(format, args...), nil, nil)
}

// ===============================
// FIELDS-BASED LOGGING METHODS
// ===============================
func (l *Logger) TraceWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    l.log(TRACE, fmt.Sprintf(format, args...), fields, nil)
}

func (l *Logger) DebugWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    l.log(DEBUG, fmt.Sprintf(format, args...), fields, nil)
}

func (l *Logger) InfoWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    l.log(INFO, fmt.Sprintf(format, args...), fields, nil)
}

func (l *Logger) NoticeWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    l.log(NOTICE, fmt.Sprintf(format, args...), fields, nil)
}

func (l *Logger) WarnWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    l.log(WARN, fmt.Sprintf(format, args...), fields, nil)
}

func (l *Logger) ErrorWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    l.log(ERROR, fmt.Sprintf(format, args...), fields, nil)
}

func (l *Logger) FatalWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    l.log(FATAL, fmt.Sprintf(format, args...), fields, nil)
}

func (l *Logger) PanicWithFields(fields map[string]interface{}, format string, args ...interface{}) {
    l.log(PANIC, fmt.Sprintf(format, args...), fields, nil)
}

// ===============================
// LOGGER CREATION & CONFIGURATION
// ===============================
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
    
    if config.BufferSize > 0 {
        l.buffer = NewBufferedWriter(config.Output, config.BufferSize, config.FlushInterval)
        l.out = l.buffer
    }
    
    if config.EnableSampling && config.SamplingRate > 1 {
        l.sampler = NewSamplingLogger(l, config.SamplingRate)
    }
    
    if config.EnableRotation && config.RotationConfig != nil {
        // Handle rotation setup
        if file, ok := config.Output.(*os.File); ok {
            var err error
            l.rotation, err = NewRotatingFileWriter(file.Name(), config.RotationConfig)
            if err == nil {
                l.out = l.rotation
            }
        }
    }
    
    return l
}

// ===============================
// ENHANCED LOGGING METHODS
// ===============================
func (l *Logger) log(level Level, msg string, fields map[string]interface{}, ctx context.Context) {
    // Sampling check
    if l.sampler != nil && !l.sampler.shouldLog() {
        return
    }
    
    if level < l.config.Level {
        return
    }
    
    start := time.Now()
    
    mergedFields := make(map[string]interface{})
    for k, v := range l.fields {
        mergedFields[k] = v
    }
    for k, v := range fields {
        mergedFields[k] = v
    }
    
    entry := &LogEntry{
        Timestamp:    time.Now(),
        Level:        level,
        LevelName:    level.String(),
        Message:      msg,
        PID:          os.Getpid(),
        Fields:       mergedFields,
        Hostname:     l.config.Hostname,
        Application:  l.config.Application,
        Version:      l.config.Version,
        Environment:  l.config.Environment,
    }
    
    if ctx != nil {
        contextValues := make(map[string]string)
        
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
    
    if l.config.ShowGoroutine {
        entry.GoroutineID = getGoroutineID()
    }
    
    if l.config.EnableStackTrace && (level >= ERROR) {
        entry.StackTrace = getStackTrace(l.config.StackTraceDepth)
    }
    
    for _, hook := range l.hooks {
        hook(entry)
    }
    
    l.mu.Lock()
    defer l.mu.Unlock()
    
    formatted, err := l.formatter.Format(entry)
    if err != nil {
        if l.errorHandler != nil {
            l.errorHandler(err)
        } else {
            fmt.Fprintf(l.errOut, "Logger error: %v\n", err)
        }
        return
    }
    
    bytesWritten, err := fmt.Fprint(l.out, formatted)
    if err != nil {
        if l.errorHandler != nil {
            l.errorHandler(err)
        }
    }
    
    l.stats.Increment(level, bytesWritten)
    
    if l.metrics != nil {
        tags := map[string]string{
            "level": level.String(),
            "application": l.config.Application,
            "environment": l.config.Environment,
        }
        l.metrics.IncrementCounter(level, tags)
    }
    
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
    
    if time.Since(start) > 100*time.Millisecond {
        slowFields := map[string]interface{}{
            "logging_duration": time.Since(start).String(),
            "message_length":   len(msg),
            "fields_count":     len(mergedFields),
        }
        l.log(WARN, "Slow logging operation", slowFields, nil)
    }
}

// ===============================
// BATCH LOGGING
// ===============================
func (l *Logger) LogBatch(entries []*LogEntry) error {
    if len(entries) == 0 {
        return nil
    }
    
    var batch strings.Builder
    for _, entry := range entries {
        formatted, err := l.formatter.Format(entry)
        if err != nil {
            return err
        }
        batch.WriteString(formatted)
    }
    
    l.mu.Lock()
    defer l.mu.Unlock()
    
    _, err := fmt.Fprint(l.out, batch.String())
    return err
}

// ===============================
// AUDIT LOGGING
// ===============================
func (l *Logger) Audit(eventType, action, resource string, userID interface{}, success bool, details map[string]interface{}) {
    auditFields := map[string]interface{}{
        "audit_event_type": eventType,
        "audit_action":     action,
        "audit_resource":   resource,
        "audit_user_id":    userID,
        "audit_success":    success,
        "audit_timestamp":  time.Now().UTC().Format(time.RFC3339),
    }
    
    for k, v := range details {
        auditFields[k] = v
    }
    
    level := INFO
    if !success {
        level = WARN
    }
    
    l.log(level, fmt.Sprintf("Audit event: %s - %s", eventType, action), auditFields, nil)
}

// ===============================
// PERFORMANCE MONITORING
// ===============================
func (l *Logger) TimeOperation(operationName string, fields map[string]interface{}, operation func()) {
    start := time.Now()
    
    defer func() {
        duration := time.Since(start)
        opFields := make(map[string]interface{})
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
        
        l.log(level, fmt.Sprintf("Operation completed: %s", operationName), opFields, nil)
        
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
// COMPREHENSIVE ROTATING FILE WRITER
// ===============================
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
}

func NewRotatingFileWriter(filename string, config *RotationConfig) (*RotatingFileWriter, error) {
    r := &RotatingFileWriter{
        filename:      filename,
        compress:      config.Compress,
        compressedExt: ".gz",
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
    
    _, err = io.Copy(gzWriter, srcFile)
    if err != nil {
        return err
    }
    
    return os.Remove(srcFilename)
}

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

func (r *RotatingFileWriter) Close() error {
    r.mu.Lock()
    defer r.mu.Unlock()
    if r.file != nil {
        return r.file.Close()
    }
    return nil
}

// ===============================
// UTILITY FUNCTIONS
// ===============================
func getGoroutineID() string {
    buf := make([]byte, 64)
    buf = buf[:runtime.Stack(buf, false)]
    str := string(buf)
    if strings.HasPrefix(str, "goroutine ") {
        str = strings.TrimPrefix(str, "goroutine ")
        if space := strings.Index(str, " "); space > 0 {
            return str[:space]
        }
    }
    return "unknown"
}

func getStackTrace(depth int) string {
    pc := make([]uintptr, depth)
    n := runtime.Callers(3, pc)
    if n == 0 {
        return ""
    }
    
    pc = pc[:n]
    frames := runtime.CallersFrames(pc)
    
    var stack strings.Builder
    for {
        frame, more := frames.Next()
        stack.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line))
        if !more {
            break
        }
    }
    
    return stack.String()
}

func shortID(id string) string {
    if len(id) > 8 {
        return id[:8]
    }
    return id
}

func generateID() string {
    b := make([]byte, 8)
    rand.Read(b)
    return base64.RawURLEncoding.EncodeToString(b)
}

func getClientIP(r *http.Request) string {
    if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
        return strings.Split(ip, ",")[0]
    }
    if ip := r.Header.Get("X-Real-IP"); ip != "" {
        return ip
    }
    return r.RemoteAddr
}

func formatValue(v interface{}, maxWidth int) string {
    switch val := v.(type) {
    case string:
        return formatString(val, maxWidth)
    case error:
        return formatString(val.Error(), maxWidth)
    default:
        strVal := fmt.Sprintf("%v", v)
        return formatString(strVal, maxWidth)
    }
}

func formatString(s string, maxWidth int) string {
    if maxWidth <= 0 {
        return s
    }
    if utf8.RuneCountInString(s) > maxWidth {
        runes := []rune(s)
        if len(runes) > maxWidth {
            return string(runes[:maxWidth]) + "..."
        }
    }
    return s
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

// ===============================
// LOGGER MANAGEMENT
// ===============================
type LoggerManager struct {
    loggers    map[string]*Logger
    defaultLogger *Logger
    mu         sync.RWMutex
}

func NewLoggerManager() *LoggerManager {
    return &LoggerManager{
        loggers:       make(map[string]*Logger),
        defaultLogger: NewDefaultLogger(),
    }
}

func (lm *LoggerManager) RegisterLogger(name string, logger *Logger) {
    lm.mu.Lock()
    defer lm.mu.Unlock()
    lm.loggers[name] = logger
}

func (lm *LoggerManager) GetLogger(name string) (*Logger, bool) {
    lm.mu.RLock()
    defer lm.mu.RUnlock()
    logger, exists := lm.loggers[name]
    return logger, exists
}

func (lm *LoggerManager) SetDefaultLogger(logger *Logger) {
    lm.mu.Lock()
    defer lm.mu.Unlock()
    lm.defaultLogger = logger
}

func (lm *LoggerManager) SetAllLevels(level Level) {
    lm.mu.Lock()
    defer lm.mu.Unlock()
    
    for _, logger := range lm.loggers {
        logger.SetLevel(level)
    }
    lm.defaultLogger.SetLevel(level)
}

func (lm *LoggerManager) GetStats() map[string]interface{} {
    lm.mu.RLock()
    defer lm.mu.RUnlock()
    
    stats := make(map[string]interface{})
    stats["total_loggers"] = len(lm.loggers)
    
    loggerStats := make(map[string]interface{})
    for name, logger := range lm.loggers {
        loggerStats[name] = logger.GetStats()
    }
    stats["loggers"] = loggerStats
    
    return stats
}

// ===============================
// CONFIGURATION MANAGEMENT
// ===============================
type LogConfig struct {
    Level         string                 `json:"level" yaml:"level"`
    Format        string                 `json:"format" yaml:"format"`
    Output        string                 `json:"output" yaml:"output"`
    EnableColors  bool                   `json:"enable_colors" yaml:"enable_colors"`
    Rotation      *RotationConfig        `json:"rotation" yaml:"rotation"`
    Fields        map[string]interface{} `json:"fields" yaml:"fields"`
    Sensitive     []string               `json:"sensitive_fields" yaml:"sensitive_fields"`
}

func LoadConfigFromFile(filename string) (*LoggerConfig, error) {
    data, err := ioutil.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    
    var config LogConfig
    ext := filepath.Ext(filename)
    
    switch strings.ToLower(ext) {
    case ".json":
        if err := json.Unmarshal(data, &config); err != nil {
            return nil, err
        }
    default:
        return nil, errors.New("unsupported config file format")
    }
    
    return BuildLoggerConfig(config)
}

func BuildLoggerConfig(cfg LogConfig) (*LoggerConfig, error) {
    level, err := ParseLevel(cfg.Level)
    if err != nil {
        return nil, err
    }
    
    config := &LoggerConfig{
        Level:         level,
        EnableColors:  cfg.EnableColors,
        ShowCaller:    true,
        ShowGoroutine: true,
        ShowPID:       true,
        ShowTraceInfo: true,
        EnableStackTrace: true,
        BufferSize:    1000,
        FlushInterval: 5 * time.Second,
    }
    
    if cfg.Output != "" {
        file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            return nil, err
        }
        config.Output = file
        config.EnableRotation = true
        config.RotationConfig = cfg.Rotation
    }
    
    switch cfg.Format {
    case "json":
        config.Formatter = &JSONFormatter{
            PrettyPrint:       true,
            TimestampFormat:   "2006-01-02T15:04:05.000Z07:00",
            SensitiveFields:   cfg.Sensitive,
            MaskSensitiveData: true,
            MaskString:        "***",
        }
    case "csv":
        config.Formatter = &CSVFormatter{
            IncludeHeader:   true,
            TimestampFormat: "2006-01-02 15:04:05.000",
            FieldOrder:      []string{"timestamp", "level", "message", "file", "line"},
        }
    default:
        config.Formatter = &TextFormatter{
            EnableColors:      cfg.EnableColors,
            ShowTimestamp:     true,
            ShowCaller:        true,
            ShowGoroutine:     true,
            ShowPID:           true,
            ShowTraceInfo:     true,
            TimestampFormat:   DEFAULT_TIMESTAMP_FORMAT,
            EnableStackTrace:  true,
            SensitiveFields:   cfg.Sensitive,
            MaskSensitiveData: true,
            MaskString:        "***",
        }
    }
    
    return config, nil
}

var globalManager = NewLoggerManager()

func GetLoggerManager() *LoggerManager {
    return globalManager
}
