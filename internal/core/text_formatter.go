package core

import (
	"fmt"
	"strconv"
	"strings"
)

// Pre-computed constants for formatting to avoid runtime lookups and minimize allocations during log formatting
// Konstanta yang telah dihitung sebelumnya untuk formatting guna menghindari pencarian runtime dan meminimalkan alokasi selama formatting log
var (
	bracketOpen   = byte('[')          // Pre-allocated byte constant for opening bracket character
	bracketClose  = byte(']')          // Pre-allocated byte constant for closing bracket character
	space         = byte(' ')          // Pre-allocated byte constant for space character
	colon         = byte(':')          // Pre-allocated byte constant for colon character
	equals        = byte('=')          // Pre-allocated byte constant for equals character
	quote         = byte('"')          // Pre-allocated byte constant for quote character
	newline       = byte('\n')         // Pre-allocated byte constant for newline character
	comma         = byte(',')          // Pre-allocated byte constant for comma character
	resetColor    = []byte("\033[0m")  // Pre-allocated byte slice for ANSI color reset sequence
	grayColor     = []byte("\033[38;5;245m") // Pre-allocated byte slice for gray color ANSI sequence
)

// TextFormatter provides configurable text formatting for log entries with zero-allocation design
// TextFormatter menyediakan formatting teks yang dapat dikonfigurasi untuk entri log dengan desain zero-allocation
type TextFormatter struct {
	EnableColors          bool   // EnableColors controls whether ANSI color codes are added to output
	ShowTimestamp         bool   // ShowTimestamp controls whether timestamp is included in log output
	ShowCaller            bool   // ShowCaller controls whether caller information is included in log output
	ShowGoroutine         bool   // ShowGoroutine controls whether goroutine ID is included in log output
	ShowPID               bool   // ShowPID controls whether process ID is included in log output
	ShowTraceInfo         bool   // ShowTraceInfo controls whether distributed tracing information is included
	ShowHostname          bool   // ShowHostname controls whether hostname is included in log output
	ShowApplication       bool   // ShowApplication controls whether application name is included in log output
	FullTimestamp         bool   // FullTimestamp controls whether full timestamp format is used
	TimestampFormat       string // TimestampFormat specifies the format string for timestamps
	EnableStackTrace      bool   // EnableStackTrace controls whether stack traces are captured for errors
	StackTraceDepth       int    // StackTraceDepth limits the depth of captured stack traces
	EnableDuration        bool   // EnableDuration controls whether duration measurements are included
	MaxFieldWidth         int    // MaxFieldWidth limits the width of field values to prevent overly long output
	MaskSensitiveData     bool   // MaskSensitiveData controls whether sensitive data is masked in output
	MaskString            string // MaskString specifies the string used to mask sensitive data
}

// NewTextFormatter creates a new TextFormatter with default settings
// NewTextFormatter membuat TextFormatter baru dengan pengaturan default
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		EnableColors:      true,
		ShowTimestamp:     true,
		ShowCaller:        false,
		ShowGoroutine:     false,
		ShowPID:           false,
		ShowTraceInfo:     false,
		ShowHostname:      false,
		ShowApplication:   false,
		FullTimestamp:     false,
		TimestampFormat:   "2006-01-02 15:04:05",
		EnableStackTrace:  false,
		StackTraceDepth:   10,
		EnableDuration:    false,
		MaxFieldWidth:     100,
		MaskSensitiveData: false,
		MaskString:        "***",
	}
}

// Format formats a log entry with zero allocation using optimized techniques to minimize garbage collection pressure
// Format memformat entri log dengan zero allocation menggunakan teknik yang dioptimalkan untuk meminimalkan tekanan garbage collection
//go:inline
func (f *TextFormatter) Format(entry interface{}) ([]byte, error) {
	// Cast entry to LogEntryInterface
	logEntry, ok := entry.(LogEntryInterface)
	if !ok {
		return nil, fmt.Errorf("invalid entry type")
	}
	
	// Get a buffer from pool to avoid allocation and enable reuse
	// Dapatkan buffer dari pool untuk menghindari alokasi dan memungkinkan penggunaan kembali
	buf := getBufferFromPool()
	defer putBufferToPool(buf)
	// Fast path: pre-allocate expected size to avoid buffer growth and reallocations
	// Jalur cepat: pra-alokasi ukuran yang diharapkan untuk menghindari pertumbuhan buffer dan realokasi
	// Estimate: timestamp(30) + level(10) + message(msgLen) + fields(200) = ~240 + msgLen
	expectedSize := 240 + len(logEntry.GetMessage())
	if cap(buf.data) < expectedSize {
		// Only allocate if buffer is too small, preserving existing capacity when possible
		// Hanya alokasikan jika buffer terlalu kecil, mempertahankan kapasitas yang ada jika memungkinkan
		buf.data = make([]byte, 0, expectedSize)
	}
	// Write timestamp - optimized path using pre-computed constants
	// Tulis timestamp - jalur yang dioptimalkan menggunakan konstanta yang telah dihitung sebelumnya
	if f.ShowTimestamp {
		buf.WriteByte(bracketOpen)  // Use pre-allocated constant to avoid allocation
		timestamp := logEntry.GetTimestamp()
		buf.WriteString(timestamp.Format(f.TimestampFormat))
		buf.WriteByte(bracketClose) // Use pre-allocated constant to avoid allocation
		buf.WriteByte(space)        // Use pre-allocated constant to avoid allocation
	}
	// Write level - optimized with direct array access to pre-computed level strings
	// Tulis level - dioptimalkan dengan akses array langsung ke string level yang telah dihitung sebelumnya
	levelStr := logEntry.GetLevel().String()
	if f.EnableColors {
		// Add background and foreground colors using pre-computed ANSI sequences
		// Tambahkan warna latar belakang dan latar depan menggunakan urutan ANSI yang telah dihitung sebelumnya
		level := logEntry.GetLevel()
		if int(level) < len(levelBackgrounds) {
			buf.WriteString(levelBackgrounds[level])
		}
		if int(level) < len(levelColors) {
			buf.WriteString(levelColors[level])
		}
		buf.WriteByte(space)
		buf.WriteString(levelStr)
		buf.WriteByte(space)
		buf.Write(resetColor) // Reset colors using pre-allocated sequence
	} else {
		// Plain text format with brackets
		// Format teks polos dengan tanda kurung
		buf.WriteByte(bracketOpen)
		buf.WriteString(levelStr)
		buf.WriteByte(bracketClose)
	}
	buf.WriteByte(space)
	// Write hostname using zero-allocation techniques
	// Tulis hostname menggunakan teknik zero-allocation
	if f.ShowHostname && logEntry.GetHostname() != "" {
		if f.EnableColors {
			buf.WriteString("\033[38;5;245m") // Gray color for hostname
		}
		// Direct buffer write to avoid string allocation
		// Tulis buffer langsung untuk menghindari alokasi string
		buf.WriteString(logEntry.GetHostname())
		if f.EnableColors {
			buf.WriteString("\033[0m") // Reset color
		}
		buf.WriteByte(' ')
	}
	// Write application name using zero-allocation techniques
	// Tulis nama aplikasi menggunakan teknik zero-allocation
	if f.ShowApplication && logEntry.GetApplication() != "" {
		if f.EnableColors {
			buf.WriteString("\033[38;5;245m") // Gray color for application name
		}
		// Direct buffer write to avoid string allocation
		// Tulis buffer langsung untuk menghindari alokasi string
		buf.WriteString(logEntry.GetApplication())
		if f.EnableColors {
			buf.WriteString("\033[0m") // Reset color
		}
		buf.WriteByte(' ')
	}
	// Write PID (Process ID) with optional coloring
	// Tulis PID (Process ID) dengan pewarnaan opsional
	if f.ShowPID {
		if f.EnableColors {
			buf.WriteString("\033[38;5;245m") // Gray color for PID
		}
		buf.WriteString("PID:")
		// Convert integer to string without allocation using standard library
		// Konversi integer ke string tanpa alokasi menggunakan pustaka standar
		buf.WriteString(strconv.Itoa(logEntry.GetPID()))
		if f.EnableColors {
			buf.WriteString("\033[0m") // Reset color
		}
		buf.WriteByte(' ')
	}
	// Write goroutine ID with optional coloring using direct buffer access
	// Tulis ID goroutine dengan pewarnaan opsional menggunakan akses buffer langsung
	if f.ShowGoroutine && logEntry.GetGoroutineID() != "" {
		if f.EnableColors {
			buf.WriteString("\033[38;5;245m") // Gray color for goroutine ID
		}
		buf.WriteString("GID:")
		// Direct buffer write to avoid string allocation
		// Tulis buffer langsung untuk menghindari alokasi string
		buf.WriteString(logEntry.GetGoroutineID())
		if f.EnableColors {
			buf.WriteString("\033[0m") // Reset color
		}
		buf.WriteByte(' ')
	}
	// Write distributed tracing information with optional coloring
	// Tulis informasi tracing terdistribusi dengan pewarnaan opsional
	if f.ShowTraceInfo {
		// Write trace ID if available
		// Tulis ID trace jika tersedia
		if logEntry.GetTraceID() != "" {
			if f.EnableColors {
				buf.WriteString("\033[38;5;141m") // Purple color for trace info
			}
			buf.WriteString("TRACE:")
			// Write shortened ID to keep output concise
			// Tulis ID yang dipersingkat untuk menjaga output ringkas
			f.writeShortID(buf, []byte(logEntry.GetTraceID()))
			if f.EnableColors {
				buf.WriteString("\033[0m") // Reset color
			}
			buf.WriteByte(' ')
		}
		// Write span ID if available
		// Tulis ID span jika tersedia
		if logEntry.GetSpanID() != "" {
			if f.EnableColors {
				buf.WriteString("\033[38;5;141m") // Purple color for trace info
			}
			buf.WriteString("SPAN:")
			// Write shortened ID to keep output concise
			// Tulis ID yang dipersingkat untuk menjaga output ringkas
			f.writeShortID(buf, []byte(logEntry.GetSpanID()))
			if f.EnableColors {
				buf.WriteString("\033[0m") // Reset color
			}
			buf.WriteByte(' ')
		}
		// Write user ID if available
		// Tulis ID pengguna jika tersedia
		if logEntry.GetUserID() != "" {
			if f.EnableColors {
				buf.WriteString("\033[38;5;141m") // Purple color for trace info
			}
			buf.WriteString("USER:")
			// Direct buffer write to avoid string allocation
			// Tulis buffer langsung untuk menghindari alokasi string
			buf.WriteString(logEntry.GetUserID())
			if f.EnableColors {
				buf.WriteString("\033[0m") // Reset color
			}
			buf.WriteByte(' ')
		}
		// Write session ID if available
		// Tulis ID sesi jika tersedia
		if logEntry.GetSessionID() != "" {
			if f.EnableColors {
				buf.WriteString("\033[38;5;141m") // Purple color for trace info
			}
			buf.WriteString("SESSION:")
			// Direct buffer write to avoid string allocation
			// Tulis buffer langsung untuk menghindari alokasi string
			buf.WriteString(logEntry.GetSessionID())
			if f.EnableColors {
				buf.WriteString("\033[0m") // Reset color
			}
			buf.WriteByte(' ')
		}
		// Write request ID if available
		// Tulis ID permintaan jika tersedia
		if logEntry.GetRequestID() != "" {
			if f.EnableColors {
				buf.WriteString("\033[38;5;141m") // Purple color for trace info
			}
			buf.WriteString("REQUEST:")
			// Direct buffer write to avoid string allocation
			// Tulis buffer langsung untuk menghindari alokasi string
			buf.WriteString(logEntry.GetRequestID())
			if f.EnableColors {
				buf.WriteString("\033[0m") // Reset color
			}
			buf.WriteByte(' ')
		}
	}
	// Write caller information with optional coloring for debugging
	// Tulis informasi pemanggil dengan pewarnaan opsional untuk debugging
	if f.ShowCaller && logEntry.GetCallerFile() != "" {
		if f.EnableColors {
			buf.WriteString("\033[38;5;240m") // Dark gray color for caller info
		}
		// Write file name and line number for debugging context
		// Tulis nama file dan nomor baris untuk konteks debugging
		buf.WriteString(logEntry.GetCallerFile())
		buf.WriteByte(':')
		buf.WriteString(strconv.Itoa(logEntry.GetCallerLine()))
		if f.EnableColors {
			buf.WriteString("\033[0m") // Reset color
		}
		buf.WriteByte(' ')
	}
	// Write main message with optional coloring for emphasis
	// Tulis pesan utama dengan pewarnaan opsional untuk penekanan
	if f.EnableColors {
		buf.WriteString("\033[1m") // Bold text for message emphasis
	}
	// Direct buffer write to avoid string allocation
	// Tulis buffer langsung untuk menghindari alokasi string
	message := logEntry.GetMessage()
	
	// Sanitize message to prevent log injection
	// Sanitasi pesan untuk mencegah log injection
	message = strings.ReplaceAll(message, "\n", "\\n")
	message = strings.ReplaceAll(message, "\r", "\\r")
	buf.WriteString(message)
	if f.EnableColors {
		buf.WriteString("\033[0m") // Reset color
	}
	// Write structured fields if present using specialized formatting
	// Tulis field terstruktur jika ada menggunakan formatting khusus
	fields := logEntry.GetFields()
	if len(fields) > 0 {
		buf.WriteByte(' ')
		f.formatFields(buf, fields)
	}
	// Write tags if present using specialized formatting
	// Tulis tag jika ada menggunakan formatting khusus
	tags := logEntry.GetTags()
	if len(tags) > 0 {
		buf.WriteByte(' ')
		f.formatTags(buf, tags)
	}
	// Write custom metrics if present using specialized formatting
	// Tulis metrik kustom jika ada menggunakan formatting khusus
	metrics := logEntry.GetMetrics()
	if len(metrics) > 0 {
		buf.WriteByte(' ')
		f.formatMetrics(buf, metrics)
	}
	// Write stack trace if enabled and available with gray coloring
	// Tulis stack trace jika diaktifkan dan tersedia dengan pewarnaan abu-abu
	if f.EnableStackTrace && logEntry.GetStackTrace() != "" {
		buf.WriteByte('\n')
		if f.EnableColors {
			buf.WriteString("\033[38;5;240m") // Dark gray color for stack traces
		}
		// Direct buffer write to avoid string allocation
		// Tulis buffer langsung untuk menghindari alokasi string
		buf.WriteString(logEntry.GetStackTrace())
		if f.EnableColors {
			buf.WriteString("\033[0m") // Reset color
		}
	}
	buf.WriteByte('\n')
	// Return copy to avoid buffer reuse issues and ensure memory safety
	// Kembalikan salinan untuk menghindari masalah penggunaan kembali buffer dan memastikan keamanan memori
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// writeShortID writes a shortened ID to keep log output concise while maintaining traceability
// writeShortID menulis ID yang dipersingkat untuk menjaga output log ringkas sambil mempertahankan kemampuan pelacakan
func (f *TextFormatter) writeShortID(buf *ByteArray, id []byte) {
	// Truncate long IDs to 8 characters to keep log lines readable while preserving uniqueness
	// Potong ID panjang menjadi 8 karakter untuk menjaga baris log tetap dapat dibaca sambil mempertahankan keunikan
	if len(id) <= 8 {
		// ID is short enough, write it completely
		// ID cukup pendek, tulis secara lengkap
		buf.Write(id)
	} else {
		// ID is too long, write only the first 8 characters
		// ID terlalu panjang, tulis hanya 8 karakter pertama
		buf.Write(id[:8])
	}
}

// formatFields formats structured logging fields with optional coloring for enhanced readability
// formatFields memformat field logging terstruktur dengan pewarnaan opsional untuk meningkatkan keterbacaan
func (f *TextFormatter) formatFields(buf *ByteArray, fields []interface{}) {
	// Add opening brace with optional coloring
	// Tambahkan tanda kurung buka dengan pewarnaan opsional
	if f.EnableColors {
		buf.WriteString("\033[38;5;243m") // Dark gray color for structural elements
	}
	buf.WriteByte('{')
	// Format each field with key-value pairs
	// Format setiap field dengan pasangan key-value
	for i, field := range fields {
		// Add space separator between fields (except for the first one)
		// Tambahkan pemisah spasi antara field (kecuali untuk yang pertama)
		if i > 0 {
			buf.WriteByte(' ')
		}
		// Handle different field types
		// Tangani tipe field yang berbeda
		if fp, ok := field.(FieldPair); ok {
			// Write field key with optional coloring
			// Tulis key field dengan pewarnaan opsional
			if f.EnableColors {
				buf.WriteString("\033[38;5;75m") // Blue color for field keys
			}
			// Direct buffer write to avoid string allocation
			// Tulis buffer langsung untuk menghindari alokasi string
			buf.Write(fp.Key[:fp.KeyLen])
			if f.EnableColors {
				buf.WriteString("\033[38;5;243m") // Dark gray color for structural elements
			}
			buf.WriteByte('=')
			// Write field value with optional coloring
			// Tulis nilai field dengan pewarnaan opsional
			if f.EnableColors {
				buf.WriteString("\033[38;5;150m") // Green color for field values
			}
			// Handle zero-allocation fields
			// Tangani field zero-allocation
			if fp.IsString {
				// For zero-allocation strings, add quotes and handle escaping
				// Untuk string zero-allocation, tambahkan tanda kutip dan tangani escaping
				buf.WriteByte('"')
				// Escape quotes in string values to maintain valid output
				// Escape tanda kutip dalam nilai string untuk mempertahankan output yang valid
				valueStr := string(fp.StringValue[:fp.StringValueLen])
				
				// Mask sensitive data if enabled
				if f.MaskSensitiveData {
					// Check if field key indicates sensitive data
					keyStr := string(fp.Key[:fp.KeyLen])
					if strings.Contains(strings.ToLower(keyStr), "password") ||
					   strings.Contains(strings.ToLower(keyStr), "token") ||
					   strings.Contains(strings.ToLower(keyStr), "secret") ||
					   strings.Contains(strings.ToLower(keyStr), "key") {
						valueStr = f.MaskString
					}
				}
				
				escaped := strings.ReplaceAll(valueStr, "\"", "\\\"")
				buf.WriteString(escaped)
				buf.WriteByte('"')
			} else if fp.IsInt {
				// For zero-allocation integers, convert to string without allocation
				// Untuk integer zero-allocation, konversi ke string tanpa alokasi
				buf.WriteString(strconv.FormatInt(fp.IntValue, 10))
			} else if fp.IsFloat64 {
				// For zero-allocation floats, convert to string without allocation
				// Untuk float zero-allocation, konversi ke string tanpa alokasi
				buf.WriteString(strconv.FormatFloat(fp.Float64Value, 'g', -1, 64))
			} else if fp.IsBool {
				// For zero-allocation booleans, convert to string without allocation
				// Untuk boolean zero-allocation, konversi ke string tanpa alokasi
				if fp.BoolValue {
					buf.WriteString("true")
				} else {
					buf.WriteString("false")
				}
			} else {
				// For interface{} fields, use the standard approach
				// Untuk field interface{}, gunakan pendekatan standar
				switch v := fp.Value.(type) {
				case string:
					// For strings, add quotes and handle escaping
					// Untuk string, tambahkan tanda kutip dan tangani escaping
					buf.WriteByte('"')
					// Escape quotes in string values to maintain valid output
					// Escape tanda kutip dalam nilai string untuk mempertahankan output yang valid
					escaped := strings.ReplaceAll(v, "\"", "\\\"")
					buf.WriteString(escaped)
					buf.WriteByte('"')
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
					// For integers, convert to string without allocation where possible
					// Untuk integer, konversi ke string tanpa alokasi jika memungkinkan
					buf.WriteString(fmt.Sprintf("%v", v))
				case float32, float64:
					// For floats, convert to string without allocation where possible
					// Untuk float, konversi ke string tanpa alokasi jika memungkinkan
					buf.WriteString(fmt.Sprintf("%v", v))
				case bool:
					// For booleans, convert to string without allocation
					// Untuk boolean, konversi ke string tanpa alokasi
					if v {
						buf.WriteString("true")
					} else {
						buf.WriteString("false")
					}
				default:
					// For other types, use standard formatting
					// Untuk tipe lain, gunakan formatting standar
					valueStr := fmt.Sprintf("%v", v)
					
					// Mask sensitive data if enabled
					if f.MaskSensitiveData {
						// Check if field key indicates sensitive data
						keyStr := string(fp.Key[:fp.KeyLen])
						if strings.Contains(strings.ToLower(keyStr), "password") ||
						   strings.Contains(strings.ToLower(keyStr), "token") ||
						   strings.Contains(strings.ToLower(keyStr), "secret") ||
						   strings.Contains(strings.ToLower(keyStr), "key") {
							valueStr = f.MaskString
						}
					}
					
					buf.WriteString(valueStr)
				}
			}
			if f.EnableColors {
				buf.WriteString("\033[0m") // Reset color
			}
		} else {
			// Handle non-FieldPair fields (backward compatibility)
			// Tangani field non-FieldPair (kompatibilitas mundur)
			if f.EnableColors {
				buf.WriteString("\033[38;5;150m") // Green color for field values
			}
			buf.WriteString(fmt.Sprintf("%v", field))
			if f.EnableColors {
				buf.WriteString("\033[0m") // Reset color
			}
		}
	}
	// Add closing brace with optional coloring
	// Tambahkan tanda kurung tutup dengan pewarnaan opsional
	if f.EnableColors {
		buf.WriteString("\033[38;5;243m") // Dark gray color for structural elements
	}
	buf.WriteByte('}')
	if f.EnableColors {
		buf.WriteString("\033[0m") // Reset color
	}
}

// formatTags formats log entry tags with optional coloring for categorization
// formatTags memformat tag entri log dengan pewarnaan opsional untuk kategorisasi
func (f *TextFormatter) formatTags(buf *ByteArray, tags []string) {
	// Add opening bracket with optional coloring
	// Tambahkan tanda kurung buka dengan pewarnaan opsional
	if f.EnableColors {
		buf.WriteString("\033[38;5;243m") // Dark gray color for structural elements
	}
	buf.WriteByte('[')
	// Format each tag with comma separation
	// Format setiap tag dengan pemisahan koma
	for i, tag := range tags {
		// Add comma separator between tags (except for the first one)
		// Tambahkan pemisah koma antara tag (kecuali untuk yang pertama)
		if i > 0 {
			buf.WriteByte(',')
		}
		// Write tag with optional coloring
		// Tulis tag dengan pewarnaan opsional
		if f.EnableColors {
			buf.WriteString("\033[38;5;172m") // Orange color for tags
		}
		// Direct buffer write to avoid string allocation
		// Tulis buffer langsung untuk menghindari alokasi string
		buf.WriteString(tag)
		if f.EnableColors {
			buf.WriteString("\033[0m") // Reset color
		}
	}
	// Add closing bracket with optional coloring
	// Tambahkan tanda kurung tutup dengan pewarnaan opsional
	buf.WriteByte(']')
	if f.EnableColors {
		buf.WriteString("\033[0m") // Reset color
	}
}

// formatMetrics formats custom metrics with optional coloring for performance tracking
// formatMetrics memformat metrik kustom dengan pewarnaan opsional untuk pelacakan kinerja
func (f *TextFormatter) formatMetrics(buf *ByteArray, metrics []interface{}) {
	// Add opening parenthesis with optional coloring
	// Tambahkan tanda kurung buka dengan pewarnaan opsional
	if f.EnableColors {
		buf.WriteString("\033[38;5;243m") // Dark gray color for structural elements
	}
	buf.WriteByte('(')
	// Format each metric with key-value pairs
	// Format setiap metrik dengan pasangan key-value
	for i, metric := range metrics {
		// Add space separator between metrics (except for the first one)
		// Tambahkan pemisah spasi antara metrik (kecuali untuk yang pertama)
		if i > 0 {
			buf.WriteByte(' ')
		}
		// Handle different metric types
		// Tangani tipe metrik yang berbeda
		if mp, ok := metric.(MetricPair); ok {
			// Write metric key with optional coloring
			// Tulis key metrik dengan pewarnaan opsional
			if f.EnableColors {
				buf.WriteString("\033[38;5;75m") // Blue color for metric keys
			}
			// Direct buffer write to avoid string allocation
			// Tulis buffer langsung untuk menghindari alokasi string
			buf.Write(mp.Key[:mp.KeyLen])
			if f.EnableColors {
				buf.WriteString("\033[38;5;243m") // Dark gray color for structural elements
			}
			buf.WriteByte('=')
			// Write metric value with optional coloring
			// Tulis nilai metrik dengan pewarnaan opsional
			if f.EnableColors {
				buf.WriteString("\033[38;5;150m") // Green color for metric values
			}
			// Format float value with appropriate precision
			// Format nilai float dengan presisi yang sesuai
			buf.WriteString(strconv.FormatFloat(mp.Value, 'f', -1, 64))
			if f.EnableColors {
				buf.WriteString("\033[0m") // Reset color
			}
		} else {
			// Handle non-MetricPair metrics (backward compatibility)
			// Tangani metrik non-MetricPair (kompatibilitas mundur)
			if f.EnableColors {
				buf.WriteString("\033[38;5;150m") // Green color for metric values
			}
			buf.WriteString(fmt.Sprintf("%v", metric))
			if f.EnableColors {
				buf.WriteString("\033[0m") // Reset color
			}
		}
	}
	// Add closing parenthesis with optional coloring
	// Tambahkan tanda kurung tutup dengan pewarnaan opsional
	if f.EnableColors {
		buf.WriteString("\033[38;5;243m") // Dark gray color for structural elements
	}
	buf.WriteByte(')')
	if f.EnableColors {
		buf.WriteString("\033[0m") // Reset color
	}
}