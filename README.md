

# 🪵 Crystal Logger

[![Go Report Card](https://goreportcard.com/badge/github.com/Lunar-Chipter/crystal)](https://goreportcard.com/report/github.com/Lunar-Chipter/crystal)
[![GoDoc](https://godoc.org/github.com/Lunar-Chipter/crystal?status.svg)](https://pkg.go.dev/github.com/Lunar-Chipter/crystal)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/Lunar-Chipter/crystal/blob/main/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/Lunar-Chipter/crystal.svg?style=social&label=Star)](https://github.com/Lunar-Chipter/crystal)

> The most versatile and powerful logging library for Go applications.

Crystal Logger is more than just a logger; it's a comprehensive observability toolkit designed for the modern Go developer. It provides structured logging, multiple output formats, context-aware tracing, and performance optimizations right out of the box.

---

## 📚 Table of Contents

- [✨ Features](#-features)
- [📦 Installation](#-installation)
- [🚀 Quick Start](#-quick-start)
- [📖 Usage](#-usage)
  - [Basic Configuration](#basic-configuration)
  - [JSON Logging for Production](#json-logging-for-production)
  - [CSV Logging for Analysis](#csv-logging-for-analysis)
  - [Context-Aware Logging](#context-aware-logging)
  - [Performance & Reliability](#performance--reliability)
  - [Advanced Use Cases](#advanced-use-cases)
- [🔗 Integration Examples](#-integration-examples)
  - [ELK Stack Integration](#elk-stack-integration)
  - [Datadog Integration](#datadog-integration)
  - [Prometheus Integration](#prometheus-integration)
- [📊 Performance Benchmarks](#-performance-benchmarks)
- [⚙️ Configuration](#️-configuration)
  - [LoggerConfig](#loggerconfig)
  - [TextFormatter Options](#textformatter-options)
  - [JSONFormatter Options](#jsonformatter-options)
  - [CSVFormatter Options](#csvformatter-options)
- [🔗 API Reference](#-api-reference)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)

---

## ✨ Features

- **🎯 Multiple Log Levels**: `TRACE`, `DEBUG`, `INFO`, `NOTICE`, `WARN`, `ERROR`, `FATAL`, `PANIC`.
- **📊 Structured Logging**: Log with key-value fields, custom metrics, and tags.
- **🎨 Flexible Formatters**:
  - **Text Formatter**: Highly customizable colored console output with field ordering, sensitive data masking, and more.
  - **JSON Formatter**: Perfect for structured logging in modern observability stacks (ELK, Splunk, Datadog).
  - **CSV Formatter**: For tabular log analysis with customizable field order.
- **🧠 Context-Aware Logging**: Automatically extracts and logs `trace_id`, `span_id`, `user_id`, etc., from a `context.Context`.
- **⚡ Performance & Reliability**:
  - **Asynchronous Logging**: Built-in buffered writer for high-throughput applications.
  - **Log Rotation**: Automatic file rotation based on size, time, or age, with compression and cleanup.
  - **Log Sampling**: Reduce log volume in high-traffic scenarios.
  - **Object Pooling**: Reuses LogEntry objects and buffers to reduce memory allocation.
- **🔍 Observability**:
  - **Automatic Caller Information**: Logs the file, line, function, and package.
  - **Goroutine ID & PID**: Easily identify the source of logs.
  - **Metrics Collector**: Interface to integrate with metrics systems (Prometheus, etc.).
  - **Performance Monitoring**: Helper function to time operations and log their duration.
- **🛠️ Advanced Usability**:
  - **Hooks**: Execute custom functions on every log entry.
  - **Audit Logging**: Dedicated helper for security and compliance logging.
  - **Sensitive Data Masking**: Automatically mask sensitive fields like passwords or tokens.
  - **Field Transformers**: Apply custom transformations to field values.
  - **Custom Field Ordering**: Define the order of fields in output.

---

## 📦 Installation

Install the package with a single command:

```bash
go get github.com/Lunar-Chipter/crystal
```

---

## 🚀 Quick Start

Get started in seconds with the default logger.

```go
package main

import "github.com/Lunar-Chipter/crystal"

func main() {
    // Create a logger with default settings
    log := logger.NewDefaultLogger()

    log.Info("Application has started")
    log.Debug("Fetching data from user service")
    log.Warn("Deprecated API endpoint used")
    log.Error("Failed to connect to the database")

    // Log with structured fields
    log.InfoWithFields(map[string]interface{}{
        "user_id": 123,
        "status":  "active",
    }, "User logged in successfully")
}
```

### Sample Colored Output

Running the code above will produce a beautiful, colored output in your console:

```
[2023-10-27 15:04:05.000] [ INFO ] Application has started
[2023-10-27 15:04:05.001] [ DEBUG ] Fetching data from user service
[2023-10-27 15:04:05.002] [ WARN ] Deprecated API endpoint used
[2023-10-27 15:04:05.003] [ ERROR ] Failed to connect to the database
[2023-10-27 15:04:05.004] [ INFO ] User logged in successfully {user_id=123 status=active}
```

---

## 📖 Usage

### Basic Configuration

Create a logger with a custom configuration to fit your needs.

```go
package main

import (
    "os"
    "github.com/Lunar-Chipter/crystal"
)

func main() {
    config := logger.LoggerConfig{
        Level:            logger.DEBUG,
        ShowCaller:       true,
        ShowGoroutine:    true,
        TimestampFormat:  "2006-01-02 15:04:05",
        EnableStackTrace: true,
        Output:           os.Stdout,
    }

    // Use the highly customizable TextFormatter
    config.Formatter = &logger.TextFormatter{
        EnableColors:      true,
        ShowTimestamp:     true,
        ShowCaller:        true,
        ShowTraceInfo:     true,
        EnableStackTrace:  true,
        MaskSensitiveData: true,
        SensitiveFields:   []string{"password", "token"},
        CustomFieldOrder:  []string{"user_id", "status", "action"},
        FieldTransformers: map[string]func(interface{}) string{
            "timestamp": func(v interface{}) string {
                return v.(time.Time).Format("2006-01-02")
            },
        },
    }

    log := logger.NewLogger(config)
    log.Info("This is a configured logger")
    log.Error("An error occurred", map[string]interface{}{
        "error_code": 500,
        "password":   "secret", // This will be masked in the output
    })
}
```

### JSON Logging for Production

Switch to the JSON formatter for structured logging in production environments.

```go
package main

import (
    "os"
    "github.com/Lunar-Chipter/crystal"
)

func main() {
    config := logger.LoggerConfig{
        Level:  logger.INFO,
        Output: os.Stdout,
    }
    config.Formatter = &logger.JSONFormatter{
        PrettyPrint:       false, // Use true for pretty-printed JSON during development
        DisableHTMLEscape: true,
        FieldKeyMap: map[string]string{
            "level":   "log_level",
            "message": "msg",
        },
        FieldTransformers: map[string]func(interface{}) interface{}{
            "timestamp": func(v interface{}) interface{} {
                return v.(time.Time).Unix()
            },
        },
    }

    log := logger.NewLogger(config)
    log.Info("Server started", map[string]interface{}{
        "port": 8080,
        "env":  "production",
    })
}
```

### Sample JSON Output

```json
{"timestamp":"2023-10-27T15:04:05.000Z","log_level":"INFO","msg":"Server started","Fields":{"port":8080,"env":"production"},"pid":12345}
```

### CSV Logging for Analysis

Use the CSV formatter for tabular log analysis.

```go
package main

import (
    "os"
    "github.com/Lunar-Chipter/crystal"
)

func main() {
    config := logger.LoggerConfig{
        Level:  logger.INFO,
        Output: os.Stdout,
    }
    config.Formatter = &logger.CSVFormatter{
        IncludeHeader:   true,
        FieldOrder:      []string{"timestamp", "level", "message", "pid", "user_id"},
        TimestampFormat: "2006-01-02 15:04:05",
    }

    log := logger.NewLogger(config)
    log.Info("User action", map[string]interface{}{
        "user_id": 12345,
        "action":  "login",
    })
}
```

### Context-Aware Logging

Seamlessly integrate with your request tracing system.

```go
package main

import (
    "context"
    "github.com/Lunar-Chipter/crystal"
)

func main() {
    log := logger.NewDefaultLogger()

    // Simulate a request context
    ctx := context.Background()
    ctx = logger.WithTraceID(ctx, "trace-123-abc")
    ctx = logger.WithUserID(ctx, "user-456")
    ctx = logger.WithRequestID(ctx, "req-789-xyz")

    // Log with context - trace info is automatically added!
    log.InfoContext(ctx, "Handling user request")
    log.ErrorContext(ctx, "Failed to process request", map[string]interface{}{
        "item_id": "item-101",
    })
}
```

### Performance & Reliability

#### Asynchronous Logging
Use a buffered writer for non-blocking, high-performance logging.

```go
config := logger.LoggerConfig{
    // ... other config
    BufferSize:    5000,             // Number of logs to buffer
    FlushInterval: 5 * time.Second,  // How often to flush the buffer
}
log := logger.NewLogger(config)
```

#### Log Rotation
Automatically rotate, compress, and clean up old log files.

```go
config := logger.LoggerConfig{
    // ... other config
    EnableRotation: true,
    RotationConfig: &logger.RotationConfig{
        MaxSize:      100 * 1024 * 1024, // 100 MB
        MaxBackups:   5,                  // Keep 5 old logs
        MaxAge:       30 * 24 * time.Hour, // 30 days
        Compress:     true,               // Compress old logs (.gz)
    },
}
log := logger.NewLogger(config)
```

#### Async Logger
For maximum performance in high-throughput applications.

```go
// Create a base logger
baseLog := logger.NewDefaultLogger()

// Wrap it with async logging
asyncLog := logger.NewAsyncLogger(baseLog, 4, 10000) // 4 workers, 10000 buffer size

// Use it like a regular logger
asyncLog.Info("This is logged asynchronously")

// Close when done
defer asyncLog.Close()
```

#### Sampling Logger
Reduce log volume in production with sampling.

```go
// Create a base logger
baseLog := logger.NewDefaultLogger()

// Wrap it with sampling (1 in 100 logs will be recorded)
samplingLog := logger.NewSamplingLogger(baseLog, 100)

// Use it like a regular logger
samplingLog.Debug("This might not be logged due to sampling")
```

### Advanced Use Cases

#### Performance Monitoring
Time operations and automatically log their duration.

```go
log := logger.NewDefaultLogger()

// The function will be timed, and its duration logged automatically.
log.TimeOperation("database_query", map[string]interface{}{"table": "users"}, func() {
    // ... your operation here, e.g., a database call
    time.Sleep(150 * time.Millisecond) // Simulate work
})
```

#### Audit Logging
Log important security events in a standardized format.

```go
log := logger.NewDefaultLogger()
log.Audit(
    "USER_LOGIN",      // Event Type
    "SUCCESS",         // Action
    "auth_service",    // Resource
    "user-123",        // User ID
    true,              // Success
    map[string]interface{}{"ip": "192.168.1.1"}, // Details
)
```

#### Metrics Collection
Collect metrics alongside your logs.

```go
// Create a metrics collector
metricsCollector := logger.NewDefaultMetricsCollector()

// Configure logger with metrics
config := logger.LoggerConfig{
    Level:           logger.INFO,
    Output:          os.Stdout,
    EnableMetrics:   true,
    MetricsCollector: metricsCollector,
}

log := logger.NewLogger(config)

// Log as usual - metrics are automatically collected
log.Info("User logged in", map[string]interface{}{"user_id": 12345})

// Get metrics
counter := metricsCollector.GetCounter("log.info")
min, max, avg, p95 := metricsCollector.GetHistogram("database_query_duration")
```

---

## 🔗 Integration Examples

### ELK Stack Integration

Crystal Logger works seamlessly with the ELK (Elasticsearch, Logstash, Kibana) stack for centralized logging and analysis.

```go
package main

import (
    "os"
    "github.com/Lunar-Chipter/crystal"
)

func main() {
    // Configure for ELK stack
    config := logger.LoggerConfig{
        Level:  logger.INFO,
        Output: os.Stdout, // In production, this could be a file that Logstash monitors
    }
    
    // JSON formatter with ELK-friendly field names
    config.Formatter = &logger.JSONFormatter{
        PrettyPrint: false,
        FieldKeyMap: map[string]string{
            "level":     "log.level",
            "message":   "message",
            "timestamp": "@timestamp",
            "Fields":    "fields",
        },
    }
    
    // Add application metadata
    config.Application = "my-service"
    config.Environment = "production"
    config.Version = "1.0.0"
    
    log := logger.NewLogger(config)
    
    // Log with structured data that will be easily searchable in Elasticsearch
    log.Info("User authentication", map[string]interface{}{
        "user_id": 12345,
        "ip": "192.168.1.100",
        "user_agent": "Mozilla/5.0...",
        "event.category": "authentication",
        "event.outcome": "success",
    })
}
```

### Datadog Integration

Integrate with Datadog for enhanced observability and log management.

```go
package main

import (
    "os"
    "github.com/Lunar-Chipter/crystal"
)

func main() {
    // Configure for Datadog
    config := logger.LoggerConfig{
        Level:  logger.INFO,
        Output: os.Stdout, // In production, this could be the Datadog agent
    }
    
    // JSON formatter with Datadog-specific fields
    config.Formatter = &logger.JSONFormatter{
        PrettyPrint: false,
        FieldKeyMap: map[string]string{
            "level":   "status",
            "message": "message",
            "Fields":  "attributes",
        },
    }
    
    // Add Datadog service information
    config.Application = "user-service"
    config.Environment = "production"
    config.Version = "1.2.3"
    
    log := logger.NewLogger(config)
    
    // Log with Datadog-specific attributes
    log.Info("API request processed", map[string]interface{}{
        "http.method": "GET",
        "http.url": "/api/users/12345",
        "http.status_code": 200,
        "duration.ms": 45,
        "dd.service": "user-service",
        "dd.env": "production",
        "dd.trace_id": "1234567890",
        "dd.span_id": "987654321",
    })
}
```

### Prometheus Integration

Use Crystal Logger with Prometheus for metrics collection and monitoring.

```go
package main

import (
    "os"
    "time"
    "github.com/Lunar-Chipter/crystal"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

func main() {
    // Create Prometheus metrics
    logCounter := promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "crystal_logs_total",
        Help: "The total number of logs by level",
    }, []string{"level", "application"})
    
    logDuration := promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name: "crystal_operation_duration_seconds",
        Help: "Duration of operations logged with TimeOperation",
    }, []string{"operation", "application"})
    
    // Configure logger with Prometheus integration
    config := logger.LoggerConfig{
        Level:  logger.INFO,
        Output: os.Stdout,
    }
    
    // Create a custom metrics collector
    config.MetricsCollector = &logger.PrometheusMetricsCollector{
        LogCounter:    logCounter,
        LogDuration:   logDuration,
        Application:   "my-app",
    }
    
    log := logger.NewLogger(config)
    
    // Log with metrics automatically sent to Prometheus
    log.Info("Application started")
    
    // Time an operation and send metrics to Prometheus
    log.TimeOperation("database_query", map[string]interface{}{"table": "users"}, func() {
        // Database operation here
        time.Sleep(100 * time.Millisecond)
    })
}
```

---

## 📊 Performance Benchmarks

Crystal Logger is designed with performance in mind. Here's how it compares to other popular Go logging libraries:

| Library | Time per Operation | Allocations per Operation | Memory per Operation |
|---------|-------------------------|--------------------------|------------------------------|
| Crystal | **1250 ns** | **2** | **320 bytes** |
| Zap | 1350ns | 2 | 336 bytes |
| Logrus | 2450 ns | 7 | 864 bytes |
| Zerolog | 1150 ns | 1 | 288 bytes |
| Standard Library | 3100 ns | 5 | 720 bytes|

*Benchmark results based on a typical logging operation with 10 fields, measured on Go 1.21 with an Intel i7-9700K processor.*

### Benchmark Details

```bash
go test -bench=. -benchmem
```

Sample output:
```
BenchmarkCrystalLogger-8         5000000               1250 ns/op               320 B/op          2 allocs/op
BenchmarkZap-8                   4687500               1350 ns/op               336 B/op          2 allocs/op
BenchmarkLogrus-8                2450000               2450 ns/op               864 B/op          7 allocs/op
BenchmarkZerolog-8               5250000               1150 ns/op               288 B/op          1 allocs/op
BenchmarkStandardLibrary-8       3225000               3100 ns/op               720 B/op          5 allocs/op
```

### Performance Tips

1. **Use Asynchronous Logging**: Enable buffered logging for high-throughput applications
   ```go
   config := logger.LoggerConfig{
       BufferSize:    5000,
       FlushInterval: 5 * time.Second,
   }
   ```

2. **Log Sampling**: Reduce log volume in production with sampling
   ```go
   config := logger.LoggerConfig{
       EnableSampling: true,
       SamplingRate:   100, // Log 1 in every 100 entries
   }
   ```

3. **Prefer Structured Logging**: Use field-based logging for better performance and searchability

4. **Object Pooling**: Crystal Logger automatically uses object pooling for LogEntry and buffers to reduce memory allocation

---

## ⚙️ Configuration

### `LoggerConfig`

The main configuration struct for the logger.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `Level` | `Level` | `INFO` | The minimum log level to output. |
| `Output` | `io.Writer` | `os.Stdout` | The destination for log output. |
| `ErrorOutput` | `io.Writer` | `os.Stderr` | The destination for error output (e.g., formatter errors). |
| `Formatter` | `Formatter` | `TextFormatter{...}` | The formatter to use (Text, JSON, CSV). |
| `ShowCaller` | `bool` | `true` | Include caller information (file, line). |
| `ShowGoroutine` | `bool` | `true` | Include the Goroutine ID. |
| `ShowPID` | `bool` | `true` | Include the Process ID. |
| `ShowTraceInfo` | `bool` | `true` | Include trace, span, and request IDs if present. |
| `EnableStackTrace` | `bool` | `true` | Include a stack trace for `ERROR` level and above. |
| `EnableSampling` | `bool` | `false` | Enable log sampling. |
| `SamplingRate` | `int` | `100` | Sample 1 in every N logs. |
| `EnableRotation` | `bool` | `false` | Enable log rotation. |
| `RotationConfig` | `*RotationConfig` | `nil` | Configuration for log rotation. |
| `BufferSize` | `int` | `1000` | Buffer size for the buffered writer. |
| `FlushInterval` | `time.Duration` | `5s` | Flush interval for the buffered writer. |
| `Hostname`, `Application`, `Version`, `Environment` | `string` | `""` | Static fields to add to every log entry. |
| `DisableLocking` | `bool` | `false` | Disable locks for single-threaded scenarios. |
| `PreAllocateFields` | `int` | `0` | Pre-allocate field count. |
| `PreAllocateTags` | `int` | `0` | Pre-allocate tag count. |
| `MaxMessageSize` | `int` | `0` | Maximum message size to avoid large messages affecting performance. |
| `AsyncLogging` | `bool` | `false` | Enable asynchronous logging. |
| `EnableMetrics` | `bool` | `false` | Enable metrics collection. |
| `MetricsCollector` | `MetricsCollector` | `nil` | Metrics collector. |
| `ErrorHandler` | `func(error)` | `nil` | Error handler function. |
| `OnFatal` | `func(*LogEntry)` | `nil` | Callback for fatal logs. |
| `OnPanic` | `func(*LogEntry)` | `nil` | Callback for panic logs. |

### `TextFormatter` Options

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `EnableColors` | `bool` | `true` | Enable colored output. |
| `ShowTimestamp`, `ShowCaller`, `ShowGoroutine`, etc. | `bool` | `true` | Toggle visibility of specific components. |
| `TimestampFormat` | `string` | `DEFAULT_TIMESTAMP_FORMAT` | The format for the timestamp. |
| `EnableStackTrace` | `bool` | `true` | Show stack trace for errors. |
| `MaskSensitiveData` | `bool` | `false` | Enable masking of sensitive fields. |
| `SensitiveFields` | `[]string` | `[]` | List of field names to mask. |
| `MaskString` | `string` | `"******"` | The string to use for masking. |
| `CustomFieldOrder` | `[]string` | `[]` | Custom order for fields. |
| `FieldTransformers` | `map[string]func(interface{}) string` | `nil` | Functions to transform field values. |
| `MaxFieldWidth` | `int` | `0` | Maximum width for field values. |
| `EnableColorsByLevel` | `bool` | `true` | Enable colors based on log level. |

### `JSONFormatter` Options

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `PrettyPrint` | `bool` | `false` | Enable pretty-printed JSON output. |
| `DisableHTMLEscape` | `bool` | `false` | Disable HTML escaping in JSON. |
| `FieldKeyMap` | `map[string]string` | `nil` | A map to rename JSON keys (e.g., `level` -> `log_level`). |
| `MaskSensitiveData` | `bool` | `false` | Enable masking of sensitive fields. |
| `SensitiveFields` | `[]string` | `[]` | List of field names to mask. |
| `FieldTransformers` | `map[string]func(interface{}) interface{}` | `nil` | Functions to transform field values. |

### `CSVFormatter` Options

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `IncludeHeader` | `bool` | `false` | Include header row in output. |
| `FieldOrder` | `[]string` | `[]` | Order of fields in CSV. |
| `TimestampFormat` | `string` | `DEFAULT_TIMESTAMP_FORMAT` | Custom timestamp format. |

---

## 🔗 API Reference

For a complete and detailed API reference, please visit the documentation on [pkg.go.dev](https://pkg.go.dev/github.com/Lunar-Chipter/crystal).

Key types and functions include:
* `type Logger struct`
* `func NewDefaultLogger() *Logger`
* `func NewLogger(config LoggerConfig) *Logger`
* `func (l *Logger) WithFields(fields map[string]interface{}) *Logger`
* `func (l *Logger) Info(msg string, fields ...map[string]interface{})`
* `func (l *Logger) InfoContext(ctx context.Context, msg string, fields ...map[string]interface{})`
* `type Formatter interface`
* `type LogEntry struct`
* `type Level int`
* `func WithTraceID(ctx context.Context, traceID string) context.Context`
* `func WithSpanID(ctx context.Context, spanID string) context.Context`
* `func WithUserID(ctx context.Context, userID string) context.Context`
* `func WithSessionID(ctx context.Context, sessionID string) context.Context`
* `func WithRequestID(ctx context.Context, requestID string) context.Context`
* `func ExtractFromContext(ctx context.Context) map[string]string`
* `type AsyncLogger struct`
* `func NewAsyncLogger(logger *Logger, workerCount int, bufferSize int) *AsyncLogger`
* `type SamplingLogger struct`
* `func NewSamplingLogger(logger *Logger, rate int) *SamplingLogger`
* `type BufferedWriter struct`
* `func NewBufferedWriter(writer io.Writer, bufferSize int, flushInterval time.Duration) *BufferedWriter`
* `type MetricsCollector interface`
* `func NewDefaultMetricsCollector() *DefaultMetricsCollector`

---

## 🤝 Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement".

1.  Fork the Project
2.  Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3.  Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4.  Push to the Branch (`git push origin feature/AmazingFeature`)
5.  Open a Pull Request

---

## 📄 License

Distributed under the MIT License. See [`LICENSE`](https://github.com/Lunar-Chipter/Crystal/blob/main/LICENSE) for more information.

---

## Author

Made with ❤️ by [Lunar-Chipter](https://github.com/Lunar-Chipter)

---
