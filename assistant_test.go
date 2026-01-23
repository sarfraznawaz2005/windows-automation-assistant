package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	copilot "github.com/github/copilot-sdk/go"
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

	// Test invalid config - tools enabled but empty directory
	config = DefaultConfig()
	config.Tools.Enabled = true
	config.Tools.Directory = ""
	if err := ValidateConfig(config); err == nil {
		t.Error("Config with tools enabled but empty directory should be invalid")
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

// ============ TOOLS.GO TESTS ============

// TestMapToStruct tests the generic map-to-struct conversion
func TestMapToStruct(t *testing.T) {
	// Test valid conversion
	data := map[string]interface{}{
		"city": "New York",
	}

	var params WeatherParams
	err := mapToStruct(data, &params)
	if err != nil {
		t.Fatalf("mapToStruct failed: %v", err)
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
	err := mapToStruct(data, &params)
	if err != nil {
		t.Fatalf("mapToStruct failed: %v", err)
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
	// Test with invalid target (non-pointer)
	var params WeatherParams
	data := "invalid data type"
	err := mapToStruct(data, params) // non-pointer target
	if err == nil {
		t.Error("mapToStruct should fail with non-pointer target")
	}
}

// TestValidateToolDefinition tests tool definition validation
func TestValidateToolDefinition(t *testing.T) {
	tests := []struct {
		name    string
		def     ToolDefinition
		wantErr bool
	}{
		{
			name: "valid definition",
			def: ToolDefinition{
				Name:        "test",
				Description: "Test tool",
				Handler:     "weather",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			def: ToolDefinition{
				Name:        "",
				Description: "Test tool",
				Handler:     "weather",
			},
			wantErr: true,
		},
		{
			name: "empty description",
			def: ToolDefinition{
				Name:        "test",
				Description: "",
				Handler:     "weather",
			},
			wantErr: true,
		},
		{
			name: "empty handler",
			def: ToolDefinition{
				Name:        "test",
				Description: "Test tool",
				Handler:     "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolDefinition(&tt.def)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateToolDefinition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
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

// TestGenerateMockWeather tests mock weather generation
func TestGenerateMockWeather(t *testing.T) {
	result := generateMockWeather("London")

	if result.City != "London" {
		t.Errorf("Expected city 'London', got '%s'", result.City)
	}
	if result.Unit != "Celsius" {
		t.Errorf("Expected unit 'Celsius', got '%s'", result.Unit)
	}
	if result.Condition == "" {
		t.Error("Condition should not be empty")
	}
	// Temperature should be within expected range
	if result.Temperature < 0 || result.Temperature > 50 {
		t.Errorf("Temperature out of expected range: %f", result.Temperature)
	}
}

// TestCreateToolHandler tests tool handler creation
func TestCreateToolHandler(t *testing.T) {
	// Test known handler
	handler, err := createToolHandler("weather")
	if err != nil {
		t.Fatalf("Failed to create weather handler: %v", err)
	}
	if handler == nil {
		t.Error("Handler should not be nil")
	}

	// Test unknown handler
	_, err = createToolHandler("unknown_handler")
	if err == nil {
		t.Error("Should return error for unknown handler")
	}
}

// TestWeatherToolHandler tests the weather tool handler
func TestWeatherToolHandler(t *testing.T) {
	invocation := copilot.ToolInvocation{
		ToolName: "weather",
		Arguments: map[string]interface{}{
			"city": "Paris",
		},
	}

	result, err := weatherToolHandler(invocation)
	if err != nil {
		t.Fatalf("weatherToolHandler failed: %v", err)
	}

	if result.ResultType != "success" {
		t.Errorf("Expected ResultType 'success', got '%s'", result.ResultType)
	}
	if !strings.Contains(result.TextResultForLLM, "Paris") {
		t.Errorf("Result should contain city name, got: %s", result.TextResultForLLM)
	}
	if !strings.Contains(result.SessionLog, "Paris") {
		t.Errorf("SessionLog should contain city name, got: %s", result.SessionLog)
	}
}

// TestWeatherToolHandlerInvalidParams tests weather handler with invalid parameters
func TestWeatherToolHandlerInvalidParams(t *testing.T) {
	invocation := copilot.ToolInvocation{
		ToolName:  "weather",
		Arguments: "invalid", // not a map
	}

	_, err := weatherToolHandler(invocation)
	if err == nil {
		t.Error("weatherToolHandler should fail with invalid parameters")
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

// TestLoadCustomToolsNoDirectory tests loading tools when directory doesn't exist
func TestLoadCustomToolsNoDirectory(t *testing.T) {
	config := DefaultConfig()
	config.Tools.Enabled = true
	config.Tools.Directory = "nonexistent-tools-directory-12345"

	tools, err := loadCustomTools(config)
	if err != nil {
		t.Fatalf("loadCustomTools should not error when directory doesn't exist: %v", err)
	}
	if tools != nil {
		t.Error("loadCustomTools should return nil when directory doesn't exist")
	}
}

// TestLoadCustomToolsWithValidTool tests loading a valid tool from YAML
func TestLoadCustomToolsWithValidTool(t *testing.T) {
	// Create temp directory with a valid tool
	tempDir := t.TempDir()
	toolYAML := `name: test_tool
description: A test tool
handler: weather
parameters:
  type: object
  properties:
    city:
      type: string
`
	err := os.WriteFile(filepath.Join(tempDir, "test.yaml"), []byte(toolYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test tool file: %v", err)
	}

	config := DefaultConfig()
	config.Tools.Enabled = true
	config.Tools.Directory = tempDir
	config.Tools.EnabledTools = []string{"test_tool"}

	tools, err := loadCustomTools(config)
	if err != nil {
		t.Fatalf("loadCustomTools failed: %v", err)
	}
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", tools[0].Name)
	}
}

// TestLoadToolFromFileInvalidYAML tests loading tool with invalid YAML
func TestLoadToolFromFileInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	invalidYAML := `this is not: valid: yaml: content`
	yamlPath := filepath.Join(tempDir, "invalid.yaml")
	err := os.WriteFile(yamlPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := DefaultConfig()
	_, err = loadToolFromFile(yamlPath, config)
	if err == nil {
		t.Error("loadToolFromFile should fail with invalid YAML")
	}
}

// TestLoadToolFromFileUnknownHandler tests loading tool with unknown handler
func TestLoadToolFromFileUnknownHandler(t *testing.T) {
	tempDir := t.TempDir()
	toolYAML := `name: test_tool
description: A test tool
handler: unknown_handler
`
	yamlPath := filepath.Join(tempDir, "unknown.yaml")
	err := os.WriteFile(yamlPath, []byte(toolYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := DefaultConfig()
	config.Tools.Enabled = true
	config.Tools.EnabledTools = []string{"test_tool"}

	_, err = loadToolFromFile(yamlPath, config)
	if err == nil {
		t.Error("loadToolFromFile should fail with unknown handler")
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
	if config.Tools.Directory != "user-tools" {
		t.Errorf("Expected tools directory 'user-tools', got '%s'", config.Tools.Directory)
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
	if renderer.renderer == nil {
		t.Error("Internal renderer should not be nil")
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
