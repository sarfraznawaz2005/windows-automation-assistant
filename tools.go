package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	copilot "github.com/github/copilot-sdk/go"
	"gopkg.in/yaml.v3"
)

// ToolDefinition represents a custom tool definition from YAML
type ToolDefinition struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Parameters  map[string]interface{} `yaml:"parameters"`
	Handler     string                 `yaml:"handler"`
}

// loadCustomTools loads and registers custom tools from the user-tools directory
func loadCustomTools(config *Config) ([]copilot.Tool, error) {
	if !config.Tools.Enabled {
		return nil, nil
	}

	toolsDir := config.Tools.Directory
	if toolsDir == "" {
		toolsDir = "user-tools"
	}

	// Check if directory exists
	if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
		return nil, nil // No tools directory, not an error
	}

	var tools []copilot.Tool

	// Find all YAML files in the tools directory
	yamlFiles, err := filepath.Glob(filepath.Join(toolsDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan tools directory: %w", err)
	}

	for _, yamlFile := range yamlFiles {
		tool, err := loadToolFromFile(yamlFile, config)
		if err != nil {
			return nil, fmt.Errorf("failed to load tool from %s: %w", yamlFile, err)
		}

		if tool != nil {
			tools = append(tools, *tool)
		}
	}

	return tools, nil
}

// loadToolFromFile loads a single tool from a YAML file
func loadToolFromFile(yamlFile string, config *Config) (*copilot.Tool, error) {
	// Read the YAML file
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read tool file: %w", err)
	}

	// Parse the YAML
	var def ToolDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse tool definition: %w", err)
	}

	// Validate the tool definition
	if err := validateToolDefinition(&def); err != nil {
		return nil, fmt.Errorf("invalid tool definition: %w", err)
	}

	// Check if this tool is enabled in config
	if !isToolEnabled(def.Name, config) {
		return nil, nil // Tool not enabled, skip it
	}

	// Create the tool handler
	handler, err := createToolHandler(def.Handler)
	if err != nil {
		return nil, fmt.Errorf("failed to create handler for tool %s: %w", def.Name, err)
	}

	// Create the copilot tool
	tool := copilot.Tool{
		Name:        def.Name,
		Description: def.Description,
		Parameters:  def.Parameters,
		Handler:     handler,
	}

	return &tool, nil
}

// validateToolDefinition validates a tool definition
func validateToolDefinition(def *ToolDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if def.Description == "" {
		return fmt.Errorf("tool description is required")
	}
	if def.Handler == "" {
		return fmt.Errorf("tool handler is required")
	}
	return nil
}

// isToolEnabled checks if a tool is enabled in the configuration
func isToolEnabled(toolName string, config *Config) bool {
	if !config.Tools.Enabled {
		return false
	}

	// If no specific tools are listed, all tools are enabled
	if len(config.Tools.EnabledTools) == 0 {
		return true
	}

	// Check if the tool is in the enabled list
	for _, enabledTool := range config.Tools.EnabledTools {
		if enabledTool == toolName {
			return true
		}
	}

	return false
}

// WeatherParams defines the parameters for the weather tool
type WeatherParams struct {
	City string `json:"city"`
}

// WeatherResult defines the result structure for the weather tool
type WeatherResult struct {
	City        string  `json:"city"`
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
	Unit        string  `json:"unit"`
}

// weatherToolHandler implements the weather tool functionality
func weatherToolHandler(invocation copilot.ToolInvocation) (copilot.ToolResult, error) {
	// Parse parameters
	var params WeatherParams
	if err := mapToStruct(invocation.Arguments, &params); err != nil {
		return copilot.ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	// Simulate weather data (in a real implementation, you'd call a weather API)
	result := generateMockWeather(params.City)

	// Format the result for the LLM
	textResult := fmt.Sprintf("Weather in %s: %.1f°C, %s",
		result.City, result.Temperature, result.Condition)

	return copilot.ToolResult{
		TextResultForLLM: textResult,
		ResultType:       "success",
		SessionLog:       fmt.Sprintf("Retrieved weather for %s", params.City),
	}, nil
}

// generateMockWeather creates mock weather data
func generateMockWeather(city string) WeatherResult {
	// Seed random generator for consistent results per city
	rand.Seed(time.Now().UnixNano() + int64(len(city)))

	conditions := []string{"Sunny", "Cloudy", "Partly Cloudy", "Rainy", "Overcast"}
	// Temperatures in Celsius (roughly equivalent to previous Fahrenheit range)
	temperatures := []float64{18, 21, 24, 27, 29, 16, 13}

	return WeatherResult{
		City:        city,
		Temperature: temperatures[rand.Intn(len(temperatures))],
		Condition:   conditions[rand.Intn(len(conditions))],
		Unit:        "Celsius",
	}
}

// mapToStruct is a helper to convert map to struct (simplified implementation)
func mapToStruct(data interface{}, target interface{}) error {
	// This is a simplified implementation. In production, you'd use a proper
	// JSON marshaling/unmarshaling or reflection-based solution
	if m, ok := data.(map[string]interface{}); ok {
		if city, exists := m["city"]; exists {
			if cityStr, ok := city.(string); ok {
				if params, ok := target.(*WeatherParams); ok {
					params.City = cityStr
					return nil
				}
			}
		}
	}
	return fmt.Errorf("unable to parse parameters")
}

// createToolHandler creates a tool handler function based on the handler name
func createToolHandler(handlerName string) (copilot.ToolHandler, error) {
	switch handlerName {
	case "weather":
		return weatherToolHandler, nil
	default:
		return nil, fmt.Errorf("unknown handler: %s", handlerName)
	}
}
