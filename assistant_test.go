package main

import (
	"fmt"
	"strings"
	"testing"
)

// TestConfigValidation tests the configuration validation
func TestConfigValidation(t *testing.T) {
	// Test valid config
	config := DefaultConfig()
	if err := ValidateConfig(config); err != nil {
		t.Errorf("Default config should be valid: %v", err)
	}

	// Test invalid config - empty model
	config.Model = ""
	if err := ValidateConfig(config); err == nil {
		t.Error("Config with empty model should be invalid")
	}

	// Test invalid config - empty system prompt
	config = DefaultConfig()
	config.SystemPrompt = ""
	if err := ValidateConfig(config); err == nil {
		t.Error("Config with empty system prompt should be invalid")
	}
}

// TestPathNormalization tests the path normalization functions
func TestPathNormalization(t *testing.T) {
	// Test basic normalization
	path := "test/path"
	normalized := NormalizePath(path)
	if normalized == "" {
		t.Error("Path normalization should not return empty string")
	}

	// Test environment variable expansion (if USERPROFILE is set)
	expanded := ExpandPath("%USERPROFILE%")
	if expanded == "%USERPROFILE%" {
		t.Log("USERPROFILE environment variable not set, skipping expansion test")
	}
}

// TestErrorHandling tests the error handling functions
func TestErrorHandling(t *testing.T) {
	// Test user-friendly error conversion
	err := getUserFriendlyError(fmt.Errorf("connection refused"), "test")
	if err == "" {
		t.Error("Should return user-friendly error message")
	}

	// Test that it contains expected text
	if !strings.Contains(err, "Cannot connect") {
		t.Errorf("Error message should be user-friendly, got: %s", err)
	}
}
