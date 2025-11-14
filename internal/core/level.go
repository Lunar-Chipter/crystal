// Package core provides core components for the crystal logger
package core

import (
	"fmt"
	"unsafe"
)

// Level represents the severity level of a log entry with zero-allocation design
// Level merepresentasikan tingkat keparahan entri log dengan desain zero-allocation
type Level uint8

const (
	// TRACE level for very detailed debugging information, typically used during development
	// Level TRACE untuk informasi debugging yang sangat detail, biasanya digunakan selama pengembangan
	TRACE Level = iota
	
	// DEBUG level for debugging information, useful for diagnosing problems
	// Level DEBUG untuk informasi debugging, berguna untuk mendiagnosis masalah
	DEBUG
	
	// INFO level for general information messages about application progress
	// Level INFO untuk pesan informasi umum tentang kemajuan aplikasi
	INFO
	
	// NOTICE level for normal but significant conditions that require attention
	// Level NOTICE untuk kondisi normal namun signifikan yang memerlukan perhatian
	NOTICE
	
	// WARN level for warning messages about potential issues or unusual conditions
	// Level WARN untuk pesan peringatan tentang masalah potensial atau kondisi tidak biasa
	WARN
	
	// ERROR level for error messages indicating that a function failed to complete
	// Level ERROR untuk pesan kesalahan yang menunjukkan bahwa suatu fungsi gagal diselesaikan
	ERROR
	
	// FATAL level for critical errors that cause program termination after logging
	// Level FATAL untuk kesalahan kritis yang menyebabkan program berhenti setelah logging
	FATAL
	
	// PANIC level for panic conditions that will cause the program to panic after logging
	// Level PANIC untuk kondisi panic yang akan menyebabkan program panic setelah logging
	PANIC
)

// Pre-computed string representations and colors for zero-allocation access to log levels
// Representasi string dan warna pra-komputasi untuk akses zero-allocation ke tingkat log
var (
	// levelStrings contains pre-allocated string representations of log levels for zero-allocation access
	// levelStrings berisi representasi string tingkat log yang pra-dialokasikan untuk akses zero-allocation
	levelStrings = [...]string{
		"TRACE", "DEBUG", "INFO", "NOTICE", "WARN", "ERROR", "FATAL", "PANIC",
	}
	
	// levelColors contains ANSI color codes for each log level to enable colored output
	// levelColors berisi kode warna ANSI untuk setiap tingkat log untuk mengaktifkan output berwarna
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
	
	// levelBackgrounds contains ANSI background color codes for each log level
	// levelBackgrounds berisi kode warna latar belakang ANSI untuk setiap tingkat log
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
	
	// levelMasks contains pre-computed bitmasks for each log level to enable fast level comparison without allocations
	// levelMasks berisi bitmask yang pra-dihitung untuk setiap tingkat log untuk memungkinkan perbandingan tingkat cepat tanpa alokasi
	levelMasks = [...]uint64{
		1 << TRACE, 1 << DEBUG, 1 << INFO, 1 << NOTICE,
		1 << WARN, 1 << ERROR, 1 << FATAL, 1 << PANIC,
	}
)

// String returns the string representation of the level using zero-allocation technique by accessing pre-computed array
// String mengembalikan representasi string dari level menggunakan teknik zero-allocation dengan mengakses array yang telah dihitung sebelumnya
func (l Level) String() string {
	// Zero-allocation access to pre-computed string representations
	// Akses zero-allocation ke representasi string yang telah dihitung sebelumnya
	if l < Level(len(levelStrings)) {
		return levelStrings[l]
	}
	return "UNKNOWN"
}

// ParseLevel parses a string representation into a Level constant using optimized switch-based approach to avoid allocations
// ParseLevel mem-parsing representasi string menjadi konstanta Level menggunakan pendekatan berbasis switch yang dioptimalkan untuk menghindari alokasi
func ParseLevel(levelStr string) (Level, error) {
	// Optimized parsing using length-based switch to minimize string comparisons and avoid allocations
	// Parsing yang dioptimalkan menggunakan switch berbasis panjang untuk meminimalkan perbandingan string dan menghindari alokasi
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

// Unsafe string/byte conversions for zero allocation using unsafe package to avoid memory copying
// Konversi string/byte unsafe untuk zero allocation menggunakan paket unsafe untuk menghindari penyalinan memori

// sToBytes converts string to byte slice without allocation by using unsafe operations to share memory
// sToBytes mengkonversi string ke slice byte tanpa alokasi dengan menggunakan operasi unsafe untuk berbagi memori
//go:inline
func sToBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// bToString converts byte slice to string without allocation by using unsafe operations to share memory
// bToString mengkonversi slice byte ke string tanpa alokasi dengan menggunakan operasi unsafe untuk berbagi memori
//go:inline
func bToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}