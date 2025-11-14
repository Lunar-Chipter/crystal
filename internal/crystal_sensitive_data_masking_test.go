package crystal

import (
	"bytes"
	"strings"
	"testing"
	"crystal/internal/core"
)

// TestSensitiveDataMasking tests sensitive data masking
func TestSensitiveDataMasking(t *testing.T) {
	var buf bytes.Buffer
	config := core.LoggerConfig{
		Level:     core.INFO,
		Output:    &buf,
		Formatter: &core.TextFormatter{
			MaskSensitiveData: true,
			MaskString:        "***",
		},
	}
	logger := core.NewLogger(config)

	// Test various sensitive data scenarios
	sensitiveDataTests := []struct {
		name     string
		message  string
		fields   []interface{}
		expected string
	}{
		{
			name:     "PasswordField",
			message:  "User login",
			fields:   []interface{}{"username", "john_doe", "password", "secret123"},
			expected: "***",
		},
		{
			name:     "TokenField",
			message:  "API call",
			fields:   []interface{}{"endpoint", "/api/data", "auth_token", "abc123xyz"},
			expected: "***",
		},
		{
			name:     "SecretField",
			message:  "Key rotation",
			fields:   []interface{}{"old_secret", "oldkey123", "new_secret", "newkey456"},
			expected: "***",
		},
		{
			name:     "KeyField",
			message:  "Encryption",
			fields:   []interface{}{"encryption_key", "encrypt123", "data", "sensitive"},
			expected: "***",
		},
		{
			name:     "NonSensitiveField",
			message:  "User info",
			fields:   []interface{}{"username", "john_doe", "email", "john@example.com"},
			expected: "john@example.com",
		},
	}

	for _, test := range sensitiveDataTests {
		t.Run(test.name, func(t *testing.T) {
			buf.Reset()
			logger.Info(test.message, test.fields...)
			output := buf.String()

			// Check that sensitive data is masked
			if strings.Contains(output, test.expected) && test.expected == "***" {
				// This is expected - the sensitive data should be masked
				if strings.Contains(output, "secret123") || 
				   strings.Contains(output, "abc123xyz") || 
				   strings.Contains(output, "oldkey123") || 
				   strings.Contains(output, "newkey456") || 
				   strings.Contains(output, "encrypt123") {
					t.Errorf("Sensitive data not masked in output: %s", output)
				}
			} else if !strings.Contains(output, test.expected) {
				t.Errorf("Expected data not found in output: %s", output)
			}
		})
	}
}