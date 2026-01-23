package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"windows-assistant/usertools"
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

// TestGetUserFriendlyErrorAllTypes tests all error type conversions
func TestGetUserFriendlyErrorAllTypes(t *testing.T) {
	tests := []struct {
		name        string
		errMsg      string
		context     string
		wantContain string
	}{
		{
			name:        "connection refused",
			errMsg:      "connection refused",
			context:     "test",
			wantContain: "Cannot connect",
		},
		{
			name:        "dial tcp error",
			errMsg:      "dial tcp 127.0.0.1:8080",
			context:     "test",
			wantContain: "Cannot connect",
		},
		{
			name:        "authentication error",
			errMsg:      "authentication failed",
			context:     "test",
			wantContain: "Authentication failed",
		},
		{
			name:        "unauthorized error",
			errMsg:      "unauthorized access",
			context:     "test",
			wantContain: "Authentication failed",
		},
		{
			name:        "model not found",
			errMsg:      "model gpt-5 not found",
			context:     "test",
			wantContain: "model is not available",
		},
		{
			name:        "timeout error",
			errMsg:      "request timeout exceeded",
			context:     "test",
			wantContain: "timed out",
		},
		{
			name:        "rate limit error",
			errMsg:      "rate limit exceeded",
			context:     "test",
			wantContain: "Rate limit",
		},
		{
			name:        "permission denied",
			errMsg:      "permission denied",
			context:     "test",
			wantContain: "Permission denied",
		},
		{
			name:        "access denied",
			errMsg:      "access denied to resource",
			context:     "test",
			wantContain: "Permission denied",
		},
		{
			name:        "generic error with context",
			errMsg:      "some unknown error",
			context:     "Loading config",
			wantContain: "Loading config failed",
		},
		{
			name:        "generic error without context",
			errMsg:      "some unknown error",
			context:     "",
			wantContain: "some unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUserFriendlyError(errors.New(tt.errMsg), tt.context)
			if !strings.Contains(result, tt.wantContain) {
				t.Errorf("getUserFriendlyError(%q, %q) = %q, want containing %q",
					tt.errMsg, tt.context, result, tt.wantContain)
			}
		})
	}
}

// ============ OUTPUT.GO TESTS ============

// TestSafeColor tests the color helper function
func TestSafeColor(t *testing.T) {
	// safeColor should return the color code as-is
	result := safeColor(colorRed)
	if result != colorRed {
		t.Errorf("safeColor should return color code, got: %q", result)
	}

	result = safeColor(colorYellow)
	if result != colorYellow {
		t.Errorf("safeColor should return color code, got: %q", result)
	}

	result = safeColor(colorReset)
	if result != colorReset {
		t.Errorf("safeColor should return reset code, got: %q", result)
	}
}

// TestJSONResponse tests JSON response marshaling
func TestJSONResponse(t *testing.T) {
	resp := JSONResponse{
		Success:  true,
		Response: "test response",
		Model:    "gpt-4.1",
		Tools:    []string{"weather", "shell"},
	}

	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal JSONResponse: %v", err)
	}

	// Verify JSON structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed["success"] != true {
		t.Error("success field should be true")
	}
	if parsed["response"] != "test response" {
		t.Errorf("response field mismatch, got: %v", parsed["response"])
	}
	if parsed["model"] != "gpt-4.1" {
		t.Errorf("model field mismatch, got: %v", parsed["model"])
	}
}

// TestJSONResponseError tests error JSON response
func TestJSONResponseError(t *testing.T) {
	resp := JSONResponse{
		Success: false,
		Error:   "test error",
		Model:   "gpt-4.1",
	}

	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal error JSONResponse: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed["success"] != false {
		t.Error("success field should be false")
	}
	if parsed["error"] != "test error" {
		t.Errorf("error field mismatch, got: %v", parsed["error"])
	}
	// Response should be omitted when empty
	if _, exists := parsed["response"]; exists && parsed["response"] != "" {
		t.Error("response should be omitted for error response")
	}
}

// ============ TOOLS.GO / USERTOOLS TESTS ============

// TestMapToStruct tests the usertools MapToStruct helper
func TestMapToStruct(t *testing.T) {
	type TestParams struct {
		City string `json:"city"`
	}

	data := map[string]interface{}{
		"city": "New York",
	}

	var params TestParams
	err := usertools.MapToStruct(data, &params)
	if err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	if params.City != "New York" {
		t.Errorf("Expected city 'New York', got '%s'", params.City)
	}
}

// TestMapToStructComplex tests conversion with more complex data
func TestMapToStructComplex(t *testing.T) {
	type TestParams struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
		Flag  bool   `json:"flag"`
	}

	data := map[string]interface{}{
		"name":  "test",
		"count": float64(42), // JSON numbers are float64
		"flag":  true,
	}

	var params TestParams
	err := usertools.MapToStruct(data, &params)
	if err != nil {
		t.Fatalf("MapToStruct failed: %v", err)
	}

	if params.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", params.Name)
	}
	if params.Count != 42 {
		t.Errorf("Expected count 42, got %d", params.Count)
	}
	if params.Flag != true {
		t.Errorf("Expected flag true, got %v", params.Flag)
	}
}

// TestMapToStructInvalid tests error handling for invalid data
func TestMapToStructInvalid(t *testing.T) {
	type TestParams struct {
		City string `json:"city"`
	}
	var params TestParams
	data := "invalid data type"
	err := usertools.MapToStruct(data, params) // non-pointer target
	if err == nil {
		t.Error("MapToStruct should fail with non-pointer target")
	}
}

// TestUsertoolsRegistry tests the usertools registry functions
func TestUsertoolsRegistry(t *testing.T) {
	// Test that tools are registered (weather and sum are auto-registered via init())
	tools := usertools.GetAll()
	if len(tools) < 2 {
		t.Errorf("Expected at least 2 tools (weather, sum), got %d", len(tools))
	}

	// Test Get function
	weather := usertools.Get("weather")
	if weather == nil {
		t.Error("weather tool should be registered")
	}
	if weather != nil && weather.Definition.Name != "weather" {
		t.Errorf("Expected tool name 'weather', got '%s'", weather.Definition.Name)
	}

	sum := usertools.Get("sum")
	if sum == nil {
		t.Error("sum tool should be registered")
	}

	// Test Get for non-existent tool
	unknown := usertools.Get("unknown_tool")
	if unknown != nil {
		t.Error("unknown tool should return nil")
	}

	// Test List function
	names := usertools.List()
	if len(names) < 2 {
		t.Errorf("Expected at least 2 tool names, got %d", len(names))
	}

	// Test Count function
	count := usertools.Count()
	if count < 2 {
		t.Errorf("Expected at least 2 tools, got %d", count)
	}

	// Test Stats function (new in lazy loading)
	loaded, useCount, _ := usertools.Stats("weather")
	if loaded {
		t.Error("weather tool handler should not be loaded before first use")
	}
	if useCount != 0 {
		t.Errorf("Expected use count 0, got %d", useCount)
	}
}

// TestLazyToolLoading tests that tools are lazily loaded
func TestLazyToolLoading(t *testing.T) {
	// Get tools - this should NOT load handlers
	tools := usertools.GetAll()
	if len(tools) == 0 {
		t.Fatal("No tools registered")
	}

	// Find the sum tool
	var sumTool *usertools.LazyTool
	for _, tool := range tools {
		if tool.Name == "sum" {
			sumTool = usertools.Get("sum")
			break
		}
	}
	if sumTool == nil {
		t.Fatal("sum tool not found")
	}

	// Initially, the handler should NOT be loaded
	loaded, _, _ := usertools.Stats("sum")
	if loaded {
		t.Error("sum handler should not be loaded before first invocation")
	}

	// Force unload all to ensure clean state
	usertools.ForceUnloadAll()

	// Verify unloaded
	loaded, _, _ = usertools.Stats("sum")
	if loaded {
		t.Error("sum handler should be unloaded after ForceUnloadAll")
	}
}

// TestIsToolEnabled tests tool enable checking
func TestIsToolEnabled(t *testing.T) {
	config := DefaultConfig()

	// Test with tools enabled and no specific list (all enabled)
	config.Tools.Enabled = true
	config.Tools.EnabledTools = []string{}
	if !isToolEnabled("any_tool", config) {
		t.Error("All tools should be enabled when EnabledTools is empty")
	}

	// Test with specific enabled tools
	config.Tools.EnabledTools = []string{"weather", "shell"}
	if !isToolEnabled("weather", config) {
		t.Error("weather should be enabled")
	}
	if isToolEnabled("unknown", config) {
		t.Error("unknown tool should not be enabled")
	}

	// Test with tools disabled
	config.Tools.Enabled = false
	if isToolEnabled("weather", config) {
		t.Error("No tools should be enabled when Tools.Enabled is false")
	}
}

// TestLoadCustomToolsDisabled tests loading tools when disabled
func TestLoadCustomToolsDisabled(t *testing.T) {
	config := DefaultConfig()
	config.Tools.Enabled = false

	tools, err := loadCustomTools(config)
	if err != nil {
		t.Fatalf("loadCustomTools should not error when disabled: %v", err)
	}
	if tools != nil {
		t.Error("loadCustomTools should return nil when disabled")
	}
}

// TestLoadCustomToolsEnabled tests loading tools when enabled
func TestLoadCustomToolsEnabled(t *testing.T) {
	config := DefaultConfig()
	config.Tools.Enabled = true
	config.Tools.EnabledTools = []string{} // All tools enabled

	tools, err := loadCustomTools(config)
	if err != nil {
		t.Fatalf("loadCustomTools should not error: %v", err)
	}
	if len(tools) < 2 {
		t.Errorf("Expected at least 2 tools (weather, sum), got %d", len(tools))
	}
}

// TestLoadCustomToolsFiltered tests loading only specific tools
func TestLoadCustomToolsFiltered(t *testing.T) {
	config := DefaultConfig()
	config.Tools.Enabled = true
	config.Tools.EnabledTools = []string{"weather"} // Only weather enabled

	tools, err := loadCustomTools(config)
	if err != nil {
		t.Fatalf("loadCustomTools should not error: %v", err)
	}
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool (weather only), got %d", len(tools))
	}
	if len(tools) > 0 && tools[0].Name != "weather" {
		t.Errorf("Expected tool 'weather', got '%s'", tools[0].Name)
	}
}

// ============ CLI.GO TESTS ============

// TestIsInteractiveMode tests interactive mode detection
func TestIsInteractiveMode(t *testing.T) {
	// Save original values
	origInteractive := *interactive
	origI := *i
	defer func() {
		*interactive = origInteractive
		*i = origI
	}()

	// Test both flags false
	*interactive = false
	*i = false
	if isInteractiveMode() {
		t.Error("Should not be interactive when both flags are false")
	}

	// Test -interactive flag
	*interactive = true
	*i = false
	if !isInteractiveMode() {
		t.Error("Should be interactive when -interactive is set")
	}

	// Test -i flag
	*interactive = false
	*i = true
	if !isInteractiveMode() {
		t.Error("Should be interactive when -i is set")
	}
}

// ============ CONFIG.GO TESTS ============

// TestDefaultConfig tests default config generation
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Model != "gpt-4.1" {
		t.Errorf("Expected default model 'gpt-4.1', got '%s'", config.Model)
	}
	if config.Output.Markdown != false {
		t.Error("Markdown should be disabled by default")
	}
	if config.Output.Streaming != true {
		t.Error("Streaming should be enabled by default")
	}
	if config.Output.Spinner != true {
		t.Error("Spinner should be enabled by default")
	}
	if config.Tools.Enabled != true {
		t.Error("Tools should be enabled by default")
	}
	if len(config.Tools.EnabledTools) != 0 {
		t.Error("EnabledTools should be empty by default (meaning all tools enabled)")
	}
	if config.SystemPrompt == "" {
		t.Error("SystemPrompt should not be empty")
	}
}

// TestSaveAndLoadConfig tests config file round-trip
func TestSaveAndLoadConfig(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Create and save config
	config := DefaultConfig()
	config.Model = "test-model"
	config.Output.Markdown = false

	err := SaveConfig(config, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if loaded.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", loaded.Model)
	}
	if loaded.Output.Markdown != false {
		t.Error("Markdown should be false after loading")
	}
}

// TestSaveConfigCreatesDirectory tests that SaveConfig creates parent directories
func TestSaveConfigCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "subdir", "config.yaml")

	config := DefaultConfig()
	err := SaveConfig(config, nestedPath)
	if err != nil {
		t.Fatalf("SaveConfig should create parent directories: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Error("Config file should exist after save")
	}
}

// TestLoadConfigNonExistentFile tests loading config from non-existent file
func TestLoadConfigNonExistentFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadConfig should fail for non-existent file")
	}
}

// TestLoadConfigInvalidYAML tests loading config with invalid YAML
func TestLoadConfigInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: here"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig should fail with invalid YAML")
	}
}

// TestFindConfigFile tests config file discovery
func TestFindConfigFile(t *testing.T) {
	// Test with no config file - should return empty
	// (This test assumes we're not in a directory with config.yaml)
	originalWd, _ := os.Getwd()
	tempDir := t.TempDir()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	result := findConfigFile()
	if result != "" {
		t.Errorf("Expected empty string when no config exists, got '%s'", result)
	}

	// Create config.yaml and test again
	os.WriteFile("config.yaml", []byte("model: test"), 0644)
	result = findConfigFile()
	if result != "config.yaml" {
		t.Errorf("Expected 'config.yaml', got '%s'", result)
	}
}

// TestBoolPtr tests the bool pointer helper
func TestBoolPtr(t *testing.T) {
	truePtr := boolPtr(true)
	if truePtr == nil || *truePtr != true {
		t.Error("boolPtr(true) should return pointer to true")
	}

	falsePtr := boolPtr(false)
	if falsePtr == nil || *falsePtr != false {
		t.Error("boolPtr(false) should return pointer to false")
	}
}

// ============ INTERACTIVE.GO TESTS ============

// TestIsExitCommand tests exit command detection
func TestIsExitCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"exit", true},
		{"EXIT", true},
		{"Exit", true},
		{"quit", true},
		{"QUIT", true},
		{"bye", true},
		{"BYE", true},
		{"q", true},
		{"Q", true},
		{"  exit  ", true},
		{"help", false},
		{"", false},
		{"exiting", false},
		{"quitting", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isExitCommand(tt.input)
			if result != tt.expected {
				t.Errorf("isExitCommand(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestHandleSpecialCommand tests special command handling
func TestHandleSpecialCommand(t *testing.T) {
	config := DefaultConfig()

	tests := []struct {
		input    string
		expected bool
	}{
		{"help", true},
		{"h", true},
		{"?", true},
		{"HELP", true},
		{"clear", true},
		{"cls", true},
		{"CLS", true},
		{"config", true},
		{"CONFIG", true},
		{"unknown", false},
		{"exit", false}, // exit is not a "special command", it's an exit command
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := handleSpecialCommand(tt.input, config)
			if result != tt.expected {
				t.Errorf("handleSpecialCommand(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// ============ MARKDOWN.GO TESTS ============

// TestRenderMarkdown tests basic markdown rendering
func TestRenderMarkdown(t *testing.T) {
	input := "# Hello World"
	result := RenderMarkdown(input)

	// Should contain some output (glamour transforms it)
	if result == "" {
		t.Error("RenderMarkdown should return non-empty output")
	}
}

// TestRenderMarkdownCodeBlock tests code block rendering
func TestRenderMarkdownCodeBlock(t *testing.T) {
	input := "```go\nfunc main() {}\n```"
	result := RenderMarkdown(input)

	if result == "" {
		t.Error("RenderMarkdown should handle code blocks")
	}
}

// TestRenderMarkdownList tests list rendering
func TestRenderMarkdownList(t *testing.T) {
	input := "- Item 1\n- Item 2\n- Item 3"
	result := RenderMarkdown(input)

	if result == "" {
		t.Error("RenderMarkdown should handle lists")
	}
}

// TestNewMarkdownRenderer tests renderer creation
func TestNewMarkdownRenderer(t *testing.T) {
	renderer := NewMarkdownRenderer()
	if renderer == nil {
		t.Error("NewMarkdownRenderer should not return nil")
	}
	// Test that renderer has compiled patterns
	if renderer.boldPattern == nil {
		t.Error("Bold pattern should be compiled")
	}
	if renderer.inlineCodePattern == nil {
		t.Error("Inline code pattern should be compiled")
	}
}

// TestMarkdownRendererRenderToTerminal tests direct terminal rendering
func TestMarkdownRendererRenderToTerminal(t *testing.T) {
	renderer := NewMarkdownRenderer()
	result, err := renderer.RenderToTerminal("**bold text**")
	if err != nil {
		t.Fatalf("RenderToTerminal failed: %v", err)
	}
	if result == "" {
		t.Error("RenderToTerminal should return non-empty output")
	}
}

// ============ PROGRESS.GO TESTS ============

// TestProgressIndicator tests progress indicator creation
func TestProgressIndicator(t *testing.T) {
	// Test enabled indicator
	indicator := NewProgressIndicator("Testing...", true)
	if indicator == nil {
		t.Fatal("NewProgressIndicator should not return nil")
	}
	if !indicator.enabled {
		t.Error("Indicator should be enabled when true is passed")
	}

	// Test disabled indicator
	indicator = NewProgressIndicator("Testing...", false)
	if indicator.enabled {
		t.Error("Indicator should be disabled when false is passed")
	}
}

// TestProgressIndicatorStartStop tests start/stop lifecycle
func TestProgressIndicatorStartStop(t *testing.T) {
	indicator := NewProgressIndicator("Testing...", true)

	// Start should not panic
	indicator.Start()

	// Stop should not panic (even if called multiple times)
	indicator.Stop()
	indicator.Stop()
}

// TestProgressIndicatorDisabledNoOp tests that disabled indicator is no-op
func TestProgressIndicatorDisabledNoOp(t *testing.T) {
	indicator := NewProgressIndicator("Testing...", false)

	// Start and stop should be no-ops for disabled indicator
	indicator.Start()
	indicator.Stop()
	// No panic = success
}

// TestShowToolExecution tests tool execution progress
func TestShowToolExecution(t *testing.T) {
	stopFunc := ShowToolExecution("test_tool", true)
	if stopFunc == nil {
		t.Error("ShowToolExecution should return a stop function")
	}

	// Stop should not panic
	stopFunc()
}

// TestShowToolExecutionDisabled tests disabled tool execution progress
func TestShowToolExecutionDisabled(t *testing.T) {
	stopFunc := ShowToolExecution("test_tool", false)
	if stopFunc == nil {
		t.Error("ShowToolExecution should return a stop function even when disabled")
	}

	// Stop should not panic
	stopFunc()
}
