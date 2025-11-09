---

# 🪵 Crystal Logger

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/Lunar-Chipter/Crystal.svg)](https://pkg.go.dev/github.com/Lunar-Chipter/Crystal)
[![Go Report Card](https://goreportcard.com/badge/github.com/Lunar-Chipter/Crystal)](https://goreportcard.com/report/github.com/Lunar-Chipter/Crystal)

A powerful, highly configurable, and feature-rich logging library for Go applications. Designed for modern development, it supports structured logging, multiple output formats, context-aware tracing, performance optimization, and much more.

## ✨ Features

-   **Multiple Log Levels**: `TRACE`, `DEBUG`, `INFO`, `NOTICE`, `WARN`, `ERROR`, `FATAL`, `PANIC`.
-   **Structured Logging**: Log with key-value fields, custom metrics, and tags.
-   **Flexible Formatters**:
    -   **Text Formatter**: Highly customizable colored console output with options for field ordering, sensitive data masking, and more.
    -   **JSON Formatter**: Perfect for structured logging in modern observability stacks (ELK, Splunk, Datadog).
    -   **CSV Formatter**: For tabular log analysis.
-   **Context-Aware Logging**: Automatically extracts and logs `trace_id`, `span_id`, `user_id`, etc., from a `context.Context`.
-   **Performance & Reliability**:
    -   **Asynchronous Logging**: Built-in buffered writer for high-throughput applications.
    -   **Log Rotation**: Automatic file rotation based on size, time, or age, with compression and cleanup.
    -   **Log Sampling**: Reduce log volume in high-traffic scenarios.
-   **Observability**:
    -   **Automatic Caller Information**: Logs the file, line, function, and package.
    -   **Goroutine ID & PID**: Easily identify the source of logs.
    -   **Metrics Collector**: Interface to integrate with metrics systems (Prometheus, etc.).
    -   **Performance Monitoring**: Helper function to time operations and log their duration.
-   **Advanced Usability**:
    -   **Hooks**: Execute custom functions on every log entry.
    -   **Audit Logging**: Dedicated helper for security and compliance logging.
    -   **Sensitive Data Masking**: Automatically mask sensitive fields like passwords or tokens.

## 📦 Installation

Install the package using `go get`:

```bash
go get github.com/Lunar-Chipter/crystal
```

## 🚀 Quick Start

Get started in seconds with the default logger.

```go
package main

import (
    "github.com/Lunar-Chipter/crystal"
)

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

**Sample Colored Output:**

```
[2023-10-27 15:04:05.000] [ INFO ] Application has started
[2023-10-27 15:04:05.001] [ DEBUG] Fetching data from user service
[2023-10-27 15:04:05.002] [ WARN ] Deprecated API endpoint used
[2023-10-27 15:04:05.003] [ ERROR] Failed to connect to the database
[2023-10-27 15:04:05.004] [ INFO ] User logged in successfully {user_id=123 status=active}
```

## 📖 Usage

### Basic Configuration

Create a logger with a custom configuration.

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

Switch to the JSON formatter for structured logging.

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
    }

    log := logger.NewLogger(config)

    log.Info("Server started", map[string]interface{}{
        "port": 8080,
        "env":  "production",
    })
}
```

**Sample JSON Output:**

```json
{"timestamp":"2023-10-27T15:04:05.000Z","log_level":"INFO","msg":"Server started","Fields":{"port":8080,"env":"production"},"pid":12345}
```

### Context-Aware Logging

Seamlessly integrate with your request tracing.

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

## ⚙️ Configuration

### `LoggerConfig`

The main configuration struct for the logger.

| Field | Type | Default | Description |
|---|---|---|---|
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

### `TextFormatter` Options

| Field | Type | Default | Description |
|---|---|---|---|
| `EnableColors` | `bool` | `true` | Enable colored output. |
| `ShowTimestamp`, `ShowCaller`, `ShowGoroutine`, etc. | `bool` | `true` | Toggle visibility of specific components. |
| `TimestampFormat` | `string` | `DEFAULT_TIMESTAMP_FORMAT` | The format for the timestamp. |
| `EnableStackTrace` | `bool` | `true` | Show stack trace for errors. |
| `MaskSensitiveData` | `bool` | `false` | Enable masking of sensitive fields. |
| `SensitiveFields` | `[]string` | `[]` | List of field names to mask. |
| `MaskString` | `string` | `"******"` | The string to use for masking. |

### `JSONFormatter` Options

| Field | Type | Default | Description |
|---|---|---|---|
| `PrettyPrint` | `bool` | `false` | Enable pretty-printed JSON output. |
| `DisableHTMLEscape` | `bool` | `false` | Disable HTML escaping in JSON. |
| `FieldKeyMap` | `map[string]string` | `nil` | A map to rename JSON keys (e.g., `level` -> `log_level`). |
| `MaskSensitiveData` | `bool` | `false` | Enable masking of sensitive fields. |
| `SensitiveFields` | `[]string` | `[]` | List of field names to mask. |

## 🔗 API Reference

For a complete and detailed API reference, please visit the documentation on [pkg.go.dev](https://pkg.go.dev/github.com/Lunar-Chipter/Crystal).

Key types and functions include:
-   `type Logger struct`
-   `func NewDefaultLogger() *Logger`
-   `func NewLogger(config LoggerConfig) *Logger`
-   `func (l *Logger) WithFields(fields map[string]interface{}) *Logger`
-   `func (l *Logger) Info(msg string, fields ...map[string]interface{})`
-   `func (l *Logger) InfoContext(ctx context.Context, msg string, fields ...map[string]interface{})`
-   `type Formatter interface`
-   `type LogEntry struct`
-   `type Level int`
-   `func WithTraceID(ctx context.Context, traceID string) context.Context`

## 🤝 Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1.  Fork the Project
2.  Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3.  Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4.  Push to the Branch (`git push origin feature/AmazingFeature`)
5.  Open a Pull Request

## 📄 License

Distributed under the MIT License. See [LICENSE](https://github.com/Lunar-Chipter/Crystal/blob/main/LICENSE) for more information.

## Author

Made by [Lunar-Chipter](https://github.com/Lunar-Chipter)
```
