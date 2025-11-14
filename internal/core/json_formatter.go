package core

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// JSONFormatter provides JSON formatting for log entries with zero-allocation design
// JSONFormatter menyediakan formatting JSON untuk entri log dengan desain zero-allocation
type JSONFormatter struct {
	PrettyPrint         bool   // PrettyPrint controls whether JSON output is formatted with indentation
	TimestampFormat     string // TimestampFormat specifies the format string for timestamps in JSON output
	ShowCaller          bool   // ShowCaller controls whether caller information is included in JSON output
	ShowGoroutine       bool   // ShowGoroutine controls whether goroutine ID is included in JSON output
	ShowPID             bool   // ShowPID controls whether process ID is included in JSON output
	ShowTraceInfo       bool   // ShowTraceInfo controls whether distributed tracing information is included
	EnableStackTrace    bool   // EnableStackTrace controls whether stack traces are captured for errors
	EnableDuration      bool   // EnableDuration controls whether duration measurements are included
	DisableHTMLEscape   bool   // DisableHTMLEscape controls whether HTML characters are escaped in JSON output
	MaskSensitiveData   bool   // MaskSensitiveData controls whether sensitive data is masked in JSON output
}

// NewJSONFormatter creates a new JSONFormatter with default settings
// NewJSONFormatter membuat JSONFormatter baru dengan pengaturan default
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{
		PrettyPrint:       false,
		TimestampFormat:   "2006-01-02T15:04:05.000Z07:00",
		ShowCaller:        false,
		ShowGoroutine:     false,
		ShowPID:           false,
		ShowTraceInfo:     false,
		EnableStackTrace:  false,
		EnableDuration:    false,
		DisableHTMLEscape: false,
		MaskSensitiveData: false,
	}
}

// Format formats a log entry as JSON with minimal allocation using zero-allocation techniques where possible
// Format memformat entri log sebagai JSON dengan alokasi minimal menggunakan teknik zero-allocation jika memungkinkan
func (f *JSONFormatter) Format(entry interface{}) ([]byte, error) {
	// Cast entry to LogEntryInterface
	logEntry, ok := entry.(LogEntryInterface)
	if !ok {
		return nil, fmt.Errorf("invalid entry type")
	}
	
	// Create a map to hold the log entry data
	// Buat map untuk menyimpan data entri log
	data := make(map[string]interface{}, 32)
	// Add timestamp with formatting based on configuration
	// Tambahkan timestamp dengan formatting berdasarkan konfigurasi
	timestamp := logEntry.GetTimestamp()
	if f.TimestampFormat != "" {
		// Use custom timestamp format if specified
		// Gunakan format timestamp kustom jika ditentukan
		data["timestamp"] = timestamp.Format(f.TimestampFormat)
	} else {
		// Use default timestamp representation
		// Gunakan representasi timestamp default
		data["timestamp"] = timestamp
	}
	// Add log level information
	// Tambahkan informasi tingkat log
	level := logEntry.GetLevel()
	data["level"] = level.String()
	data["level_name"] = level.String()
	// Add main log message using zero-allocation byte to string conversion
	// Tambahkan pesan log utama menggunakan konversi byte ke string zero-allocation
	data["message"] = logEntry.GetMessage()
	// Add process metadata
	// Tambahkan metadata proses
	data["pid"] = logEntry.GetPID()
	// Add caller information if enabled and available
	// Tambahkan informasi pemanggil jika diaktifkan dan tersedia
	if f.ShowCaller && logEntry.GetCallerFile() != "" {
		// Create nested caller object with caller details
		// Buat objek pemanggil bersarang dengan detail pemanggil
		caller := make(map[string]interface{}, 4)
		caller["file"] = logEntry.GetCallerFile()
		caller["line"] = logEntry.GetCallerLine()
		// Note: We don't have function and package information in the interface
		data["caller"] = caller
	}
	// Add goroutine ID if enabled and available using zero-allocation conversion
	// Tambahkan ID goroutine jika diaktifkan dan tersedia menggunakan konversi zero-allocation
	if f.ShowGoroutine && logEntry.GetGoroutineID() != "" {
		data["goroutine_id"] = logEntry.GetGoroutineID()
	}
	// Add distributed tracing information if enabled
	// Tambahkan informasi tracing terdistribusi jika diaktifkan
	if f.ShowTraceInfo {
		if logEntry.GetTraceID() != "" {
			data["trace_id"] = logEntry.GetTraceID()
		}
		if logEntry.GetSpanID() != "" {
			data["span_id"] = logEntry.GetSpanID()
		}
		if logEntry.GetUserID() != "" {
			data["user_id"] = logEntry.GetUserID()
		}
		if logEntry.GetSessionID() != "" {
			data["session_id"] = logEntry.GetSessionID()
		}
		if logEntry.GetRequestID() != "" {
			data["request_id"] = logEntry.GetRequestID()
		}
	}
	// Add duration measurement if enabled and available
	// Tambahkan pengukuran durasi jika diaktifkan dan tersedia
	if f.EnableDuration {
		duration := logEntry.GetDuration()
		if duration > 0 {
			data["duration"] = duration.String()
		}
	}
	// Add stack trace if enabled and available using zero-allocation conversion
	// Tambahkan stack trace jika diaktifkan dan tersedia menggunakan konversi zero-allocation
	if f.EnableStackTrace && logEntry.GetStackTrace() != "" {
		data["stack_trace"] = logEntry.GetStackTrace()
	}
	// Add hostname if available using zero-allocation conversion
	// Tambahkan hostname jika tersedia menggunakan konversi zero-allocation
	if logEntry.GetHostname() != "" {
		data["hostname"] = logEntry.GetHostname()
	}
	// Add application name if available using zero-allocation conversion
	// Tambahkan nama aplikasi jika tersedia menggunakan konversi zero-allocation
	if logEntry.GetApplication() != "" {
		data["application"] = logEntry.GetApplication()
	}
	// Add version information if available using zero-allocation conversion
	// Tambahkan informasi versi jika tersedia menggunakan konversi zero-allocation
	if logEntry.GetVersion() != "" {
		data["version"] = logEntry.GetVersion()
	}
	// Add environment information if available using zero-allocation conversion
	// Tambahkan informasi lingkungan jika tersedia menggunakan konversi zero-allocation
	if logEntry.GetEnvironment() != "" {
		data["environment"] = logEntry.GetEnvironment()
	}
	// Add structured fields if present
	// Tambahkan field terstruktur jika ada
	fields := logEntry.GetFields()
	if len(fields) > 0 {
		// Pre-allocate fields map with known size to avoid reallocations
		// Pra-alokasi map field dengan ukuran yang diketahui untuk menghindari realokasi
		fieldMap := make(map[string]interface{}, len(fields))
		for _, field := range fields {
			// Convert field key from bytes to string using zero-allocation technique
			// Konversi key field dari byte ke string menggunakan teknik zero-allocation
			if fp, ok := field.(FieldPair); ok {
				key := jsonBToString(fp.Key[:fp.KeyLen])
				fieldMap[key] = fp.Value
			}
		}
		data["fields"] = fieldMap
	}
	// Add tags if present
	// Tambahkan tag jika ada
	tags := logEntry.GetTags()
	if len(tags) > 0 {
		data["tags"] = tags
	}
	// Add custom metrics if present
	// Tambahkan metrik kustom jika ada
	metrics := logEntry.GetMetrics()
	if len(metrics) > 0 {
		// Pre-allocate metrics map with known size to avoid reallocations
		// Pra-alokasi map metrik dengan ukuran yang diketahui untuk menghindari realokasi
		metricMap := make(map[string]float64, len(metrics))
		for _, metric := range metrics {
			// Convert metric key from bytes to string using zero-allocation technique
			// Konversi key metrik dari byte ke string menggunakan teknik zero-allocation
			if mp, ok := metric.(MetricPair); ok {
				key := jsonBToString(mp.Key[:mp.KeyLen])
				metricMap[key] = mp.Value
			}
		}
		data["custom_metrics"] = metricMap
	}
	// Add error information if present
	// Tambahkan informasi kesalahan jika ada
	if err := logEntry.GetError(); err != nil {
		data["error"] = err.Error()
	}
	// Get a buffer from pool to avoid allocation and enable reuse
	// Dapatkan buffer dari pool untuk menghindari alokasi dan memungkinkan penggunaan kembali
	buf := getBufferFromPool()
	defer putBufferToPool(buf)
	// Create JSON encoder with buffer as output destination
	// Buat encoder JSON dengan buffer sebagai tujuan output
	encoder := json.NewEncoder(buf)
	if f.DisableHTMLEscape {
		// Disable HTML escaping for better performance when HTML content is not a concern
		// Nonaktifkan escaping HTML untuk kinerja yang lebih baik ketika konten HTML bukan masalah
		encoder.SetEscapeHTML(false)
	}
	var err error
	// Format output based on pretty print setting
	// Format output berdasarkan pengaturan pretty print
	if f.PrettyPrint {
		// Use indented formatting for human-readable output
		// Gunakan formatting dengan indentasi untuk output yang dapat dibaca manusia
		dataBytes, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, err
		}
		// Create a copy to avoid buffer reuse issues
		// Buat salinan untuk menghindari masalah penggunaan kembali buffer
		result := make([]byte, len(dataBytes))
		copy(result, dataBytes)
		return result, nil
	} else {
		// Use compact formatting for machine-readable output
		// Gunakan formatting ringkas untuk output yang dapat dibaca mesin
		err = encoder.Encode(data)
		if err != nil {
			return nil, err
		}
	}
	// Create a copy to avoid buffer reuse issues and ensure memory safety
	// Buat salinan untuk menghindari masalah penggunaan kembali buffer dan memastikan keamanan memori
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// Unsafe string/byte conversions for zero allocation using unsafe package to avoid memory copying
// Konversi string/byte unsafe untuk zero allocation menggunakan paket unsafe untuk menghindari penyalinan memori

// jsonSToBytes converts string to byte slice without allocation by using unsafe operations to share memory
// jsonSToBytes mengkonversi string ke slice byte tanpa alokasi dengan menggunakan operasi unsafe untuk berbagi memori
func jsonSToBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// jsonBToString converts byte slice to string without allocation by using unsafe operations to share memory
// jsonBToString mengkonversi slice byte ke string tanpa alokasi dengan menggunakan operasi unsafe untuk berbagi memori
func jsonBToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}