package main

import (
	"fmt"
	"os"
	"path/filepath"
	"gopkg.in/yaml.v3"
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

	// Directory containing custom tools
	Directory string `yaml:"directory" json:"directory"`

	// List of enabled tools
	EnabledTools []string `yaml:"enabled_tools" json:"enabled_tools"`
}

// OutputConfig holds output formatting configuration
type OutputConfig struct {
	// Enable markdown rendering in responses
	Markdown bool `yaml:"markdown" json:"markdown"`

	// Enable JSON output mode
	JSON bool `yaml:"json" json:"json"`
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
		SystemPrompt: `You are `Windows Automation Assistant` who can help automate various tasks on Windows 11 with various Windows, cygwin64 and other tools available at your disposal. Rules you must follow:
 (1) All user requests are about automation or working with Windows in current folder's context so never use google search (unless question is not related to automation or Windows tasks), only use shell or other tools needed to perform automation or Windows tasks to perform user request.
(2) Always show outputs of any commands you run, tools you use or any steps you perform to complete the given user request.
(3) Do NOT ask any questions, make sane assumptions on your own based on given task.
(4) STYLE: You can use markdown formatting for better readability when appropriate (tables, lists, code blocks, etc.)
(5) Always put your answer on a new line, use paragraphs.

IMPORTANT: You can use markdown formatting to make responses more readable and structured.`,
		Output: OutputConfig{
			Markdown: true,  // Enable markdown by default
			JSON:     false, // JSON output controlled by CLI flag
		},
		Tools: ToolsConfig{
			Enabled:      true,
			Directory:    "user-tools",
			EnabledTools: []string{"weather"},
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

	// If no config file found, return default
	if configPath == "" {
		return DefaultConfig(), nil
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
	// If no path specified, use default
	if configPath == "" {
		configPath = findConfigFile()
		if configPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot determine home directory: %w", err)
			}
			configPath = filepath.Join(homeDir, ".config.yaml")
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
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
func findConfigFile() string {
	// Check current directory first
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}

	// Check home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(homeDir, ".config.yaml")
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

	// Validate tools directory if enabled
	if config.Tools.Enabled && config.Tools.Directory == "" {
		return fmt.Errorf("tools.directory cannot be empty when tools are enabled")
	}

	return nil
}
