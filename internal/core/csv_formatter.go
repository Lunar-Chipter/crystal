package core

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

// CSVFormatter provides CSV formatting for log entries with zero-allocation design
// CSVFormatter menyediakan formatting CSV untuk entri log dengan desain zero-allocation
type CSVFormatter struct {
	TimestampFormat   string // TimestampFormat specifies the format string for timestamps in CSV output
	IncludeHeaders    bool   // IncludeHeaders controls whether CSV headers are included in output
	ShowCaller        bool   // ShowCaller controls whether caller information is included in CSV output
	ShowGoroutine     bool   // ShowGoroutine controls whether goroutine ID is included in CSV output
	ShowPID           bool   // ShowPID controls whether process ID is included in CSV output
	ShowTraceInfo     bool   // ShowTraceInfo controls whether distributed tracing information is included
	EnableStackTrace  bool   // EnableStackTrace controls whether stack traces are captured for errors
	EnableDuration    bool   // EnableDuration controls whether duration measurements are included
	Delimiter         rune   // Delimiter specifies the character used to separate fields in CSV output
}

// Unsafe string/byte conversions for zero allocation using unsafe package to avoid memory copying
// Konversi string/byte unsafe untuk zero allocation menggunakan paket unsafe untuk menghindari penyalinan memori
// csvSToBytes converts string to byte slice without allocation by using unsafe operations to share memory
// csvSToBytes mengkonversi string ke slice byte tanpa alokasi dengan menggunakan operasi unsafe untuk berbagi memori
func csvSToBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// Format formats a log entry as CSV with minimal allocation using zero-allocation techniques where possible
// Format memformat entri log sebagai CSV dengan alokasi minimal menggunakan teknik zero-allocation jika memungkinkan
func (f *CSVFormatter) Format(entry interface{}) ([]byte, error) {
	// Cast entry to LogEntryInterface
	logEntry, ok := entry.(LogEntryInterface)
	if !ok {
		return nil, fmt.Errorf("invalid entry type")
	}

	// Create CSV record with pre-allocated capacity for performance
	// Buat record CSV dengan kapasitas yang telah dialokasikan sebelumnya untuk kinerja
	record := make([]string, 0, 32)
	
	// Add timestamp with formatting based on configuration
	// Tambahkan timestamp dengan formatting berdasarkan konfigurasi
	if f.TimestampFormat != "" {
		// Use custom timestamp format if specified
		// Gunakan format timestamp kustom jika ditentukan
		record = append(record, logEntry.GetTimestamp().Format(f.TimestampFormat))
	} else {
		// Use default timestamp representation
		// Gunakan representasi timestamp default
		record = append(record, logEntry.GetTimestamp().String())
	}
	
	// Add log level information
	// Tambahkan informasi tingkat log
	level := logEntry.GetLevel()
	record = append(record, level.String())
	
	// Add main log message
	// Tambahkan pesan log utama
	record = append(record, logEntry.GetMessage())
	
	// Add process metadata
	// Tambahkan metadata proses
	record = append(record, strconv.Itoa(logEntry.GetPID()))
	
	// Add caller information if enabled
	// Tambahkan informasi pemanggil jika diaktifkan
	if f.ShowCaller && logEntry.GetCallerFile() != "" {
		// Add caller file and line information
		// Tambahkan informasi file dan baris pemanggil
		record = append(record, logEntry.GetCallerFile())
		record = append(record, strconv.Itoa(logEntry.GetCallerLine()))
	}
	
	// Add goroutine ID if enabled and available
	// Tambahkan ID goroutine jika diaktifkan dan tersedia
	if f.ShowGoroutine && logEntry.GetGoroutineID() != "" {
		record = append(record, logEntry.GetGoroutineID())
	}
	
	// Add distributed tracing information if enabled
	// Tambahkan informasi tracing terdistribusi jika diaktifkan
	if f.ShowTraceInfo {
		if logEntry.GetTraceID() != "" {
			record = append(record, logEntry.GetTraceID())
		}
		if logEntry.GetSpanID() != "" {
			record = append(record, logEntry.GetSpanID())
		}
		if logEntry.GetUserID() != "" {
			record = append(record, logEntry.GetUserID())
		}
		if logEntry.GetSessionID() != "" {
			record = append(record, logEntry.GetSessionID())
		}
		if logEntry.GetRequestID() != "" {
			record = append(record, logEntry.GetRequestID())
		}
	}
	
	// Add duration measurement if enabled and available
	// Tambahkan pengukuran durasi jika diaktifkan dan tersedia
	if f.EnableDuration && logEntry.GetDuration() > 0 {
		record = append(record, logEntry.GetDuration().String())
	}
	
	// Add stack trace if enabled and available
	// Tambahkan stack trace jika diaktifkan dan tersedia
	if f.EnableStackTrace && logEntry.GetStackTrace() != "" {
		record = append(record, logEntry.GetStackTrace())
	}
	
	// Add hostname if available
	// Tambahkan hostname jika tersedia
	if logEntry.GetHostname() != "" {
		record = append(record, logEntry.GetHostname())
	}
	
	// Add application name if available
	// Tambahkan nama aplikasi jika tersedia
	if logEntry.GetApplication() != "" {
		record = append(record, logEntry.GetApplication())
	}
	
	// Add version information if available
	// Tambahkan informasi versi jika tersedia
	if logEntry.GetVersion() != "" {
		record = append(record, logEntry.GetVersion())
	}
	
	// Add environment information if available
	// Tambahkan informasi lingkungan jika tersedia
	if logEntry.GetEnvironment() != "" {
		record = append(record, logEntry.GetEnvironment())
	}
	
	// Add custom fields if available
	// Tambahkan field kustom jika tersedia
	fields := logEntry.GetFields()
	for _, field := range fields {
		if fp, ok := field.(FieldPair); ok {
			key := bToString(fp.Key[:fp.KeyLen])
			value := ""
			switch v := fp.Value.(type) {
			case string:
				value = v
			case []byte:
				value = bToString(v)
			default:
				value = fmt.Sprintf("%v", v)
			}
			// Format as key=value
			// Format sebagai key=value
			record = append(record, key+"="+value)
		}
	}
	
	// Add tags if available
	// Tambahkan tag jika tersedia
	tags := logEntry.GetTags()
	if len(tags) > 0 {
		// Join tags with comma separator
		// Gabungkan tag dengan pemisah koma
		record = append(record, strings.Join(tags, ","))
	}
	
	// Add custom metrics if available
	// Tambahkan metrik kustom jika tersedia
	metrics := logEntry.GetMetrics()
	for _, metric := range metrics {
		if mp, ok := metric.(MetricPair); ok {
			key := bToString(mp.Key[:mp.KeyLen])
			// Format float value with appropriate precision
			// Format nilai float dengan presisi yang sesuai
			value := strconv.FormatFloat(mp.Value, 'f', -1, 64)
			// Format as key=value
			// Format sebagai key=value
			record = append(record, key+"="+value)
		}
	}
	
	// Add error information if available
	// Tambahkan informasi kesalahan jika tersedia
	if logEntry.GetError() != nil {
		record = append(record, logEntry.GetError().Error())
	}
	
	// Write record to CSV writer
	// Tulis record ke writer CSV
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	if err := writer.Write(record); err != nil {
		return nil, err
	}
	writer.Flush()
	
	return buf.Bytes(), writer.Error()
}