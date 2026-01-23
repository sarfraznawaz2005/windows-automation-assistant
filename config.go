package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

// Config represents the application configuration
type Config struct {
	// Model to use for Copilot sessions
	Model string `yaml:"model" json:"model"`

	// Debug mode settings
	Debug bool `yaml:"debug" json:"debug"`

	// Custom system prompt to override default
	SystemPrompt string `yaml:"system_prompt" json:"system_prompt"`

	// Output formatting options
	Output OutputConfig `yaml:"output" json:"output"`

	// Custom tools configuration
	Tools ToolsConfig `yaml:"tools" json:"tools"`

	// Copilot client options
	ClientOptions ClientConfig `yaml:"client" json:"client"`
}

// ToolsConfig holds custom tools configuration
type ToolsConfig struct {
	// Enable custom tools
	Enabled bool `yaml:"enabled" json:"enabled"`

	// List of enabled tools (empty means all tools are enabled)
	EnabledTools []string `yaml:"enabled_tools" json:"enabled_tools"`
}

// OutputConfig holds output formatting configuration
type OutputConfig struct {
	// Enable markdown rendering in responses
	Markdown bool `yaml:"markdown" json:"markdown"`

	// Enable JSON output mode
	JSON bool `yaml:"json" json:"json"`

	// Enable loading/progress spinner
	Spinner bool `yaml:"spinner" json:"spinner"`

	// Enable response streaming
	Streaming bool `yaml:"streaming" json:"streaming"`
}

// ClientConfig holds Copilot client configuration
type ClientConfig struct {
	// CLI path
	CLIPath string `yaml:"cli_path" json:"cli_path"`

	// Log level
	LogLevel string `yaml:"log_level" json:"log_level"`

	// Auto restart settings
	AutoRestart *bool `yaml:"auto_restart" json:"auto_restart"`

	// Auto start settings
	AutoStart *bool `yaml:"auto_start" json:"auto_start"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Model: "gpt-4.1",
		Debug: false,
		SystemPrompt: "You are `Windows Automation Assistant` who can help automate various tasks on Windows 11 with various Windows, cygwin64 and other tools available at your disposal. Rules you must follow:\n" +
			"(1) All user requests are about automation or working with Windows in current folder's context so never use web search unless question is not related to automation or Windows tasks. \n" +
			"(2) Always show outputs of any commands you run, tools you use or any steps you perform to complete the given user request.\n" +
			"(3) Do NOT ask any questions, make sane assumptions on your own based on given task.\n" +
			"(4) STYLE: You can use markdown formatting for better readability when appropriate (tables, lists, code blocks, etc.)\n" +
			"\nIMPORTANT: Make sure to always use available OS or any ohther tools to perform given tasks whenever possible.",
		Output: OutputConfig{
			Markdown:  false, // Disable markdown by default
			JSON:      false, // JSON output controlled by CLI flag
			Spinner:   true,  // Enable loading spinner by default
			Streaming: true,  // Enable streaming by default
		},
		Tools: ToolsConfig{
			Enabled:      true,
			EnabledTools: []string{}, // Empty means all tools are enabled
		},
		ClientOptions: ClientConfig{
			LogLevel:    "error",
			AutoRestart: boolPtr(true),
			AutoStart:   boolPtr(true),
		},
	}
}

// LoadConfig loads configuration from file or returns default
func LoadConfig(configPath string) (*Config, error) {
	// If no custom path, try default locations
	if configPath == "" {
		configPath = findConfigFile()
	}

	// If no config file found, create default config file and return default
	if configPath == "" {
		config := DefaultConfig()
		// Auto-create config file in current directory
		if err := SaveConfig(config, "config.yaml"); err != nil {
			// Non-fatal: just log if debug and continue with defaults
			if os.Getenv("ASSISTANT_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Could not auto-create config file: %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Created default config file: config.yaml\n")
		}
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Parse YAML
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	return config, nil
}

// SaveConfig saves the configuration to a file
func SaveConfig(config *Config, configPath string) error {
	// If no path specified, use current directory
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// findConfigFile looks for config file in standard locations
// Priority: 1. ./config.yaml (current directory)
//  2. ~/.assistant-config.yaml (home directory)
func findConfigFile() string {
	// Check current directory first
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}

	// Check home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(homeDir, ".assistant-config.yaml")
		if _, err := os.Stat(homeConfig); err == nil {
			return homeConfig
		}
	}

	return ""
}

// boolPtr returns a pointer to a bool value
func boolPtr(v bool) *bool {
	return &v
}

// ValidateConfig validates the configuration
func ValidateConfig(config *Config) error {
	// Validate model
	if config.Model == "" {
		return fmt.Errorf("model cannot be empty")
	}

	// Validate system prompt
	if config.SystemPrompt == "" {
		return fmt.Errorf("system_prompt cannot be empty")
	}

	return nil
}
