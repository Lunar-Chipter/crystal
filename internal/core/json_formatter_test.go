package core

import (
	"strings"
	"testing"
	"time"
)

func TestJSONFormatterFormat(t *testing.T) {
	formatter := NewJSONFormatter()
	
	// Create a properly structured LogEntry
	message := "Test message"
	var messageBuf [1024]byte
	copy(messageBuf[:], message)
	
	entry := &LogEntry{
		Timestamp:  time.Now(),
		Level:      INFO,
		MessageLen: len(message),
	}
	copy(entry.Message[:], messageBuf[:])
	
	output, err := formatter.Format(entry)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(output) == 0 {
		t.Error("Expected formatted output, got empty string")
	}
	
	outputStr := string(output)
	
	// Check that output contains some expected elements
	if !strings.Contains(outputStr, "INFO") {
		t.Error("Expected INFO level in JSON output")
	}
	
	if !strings.Contains(outputStr, message) {
		t.Error("Expected message in JSON output")
	}
}

func TestJSONFormatterWithFields(t *testing.T) {
	formatter := NewJSONFormatter()
	
	// Create a properly structured LogEntry with fields
	message := "User login"
	var messageBuf [1024]byte
	copy(messageBuf[:], message)
	
	// Create field keys
	userIDKey := "user_id"
	ipKey := "ip_address"
	var userIDKeyBuf, ipKeyBuf [64]byte
	copy(userIDKeyBuf[:], userIDKey)
	copy(ipKeyBuf[:], ipKey)
	
	entry := &LogEntry{
		Timestamp:   time.Now(),
		Level:       INFO,
		MessageLen:  len(message),
		FieldsCount: 2,
	}
	
	copy(entry.Message[:], messageBuf[:])
	
	// Set up fields properly
	entry.Fields[0] = FieldPair{
		Key:       userIDKeyBuf,
		KeyLen:    len(userIDKey),
		Value:     12345,
		IsInt:     true,
		IntValue:  12345,
	}
	
	entry.Fields[1] = FieldPair{
		Key:      ipKeyBuf,
		KeyLen:   len(ipKey),
		Value:    "192.168.1.1",
		IsString: true,
	}
	copy(entry.Fields[1].StringValue[:], "192.168.1.1")
	entry.Fields[1].StringValueLen = len("192.168.1.1")
	
	output, err := formatter.Format(entry)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(output) == 0 {
		t.Error("Expected formatted output, got empty string")
	}
	
	outputStr := string(output)
	
	// Check that output contains expected fields
	if !strings.Contains(outputStr, "user_id") || !strings.Contains(outputStr, "12345") {
		t.Error("Expected user_id field in JSON output")
	}
	
	if !strings.Contains(outputStr, "ip_address") || !strings.Contains(outputStr, "192.168.1.1") {
		t.Error("Expected ip_address field in JSON output")
	}
}