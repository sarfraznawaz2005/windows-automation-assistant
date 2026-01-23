package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	Country     string  `json:"country"`
	Temperature float64 `json:"temperature"`
	FeelsLike   float64 `json:"feels_like"`
	Humidity    string  `json:"humidity"`
	WindSpeed   string  `json:"wind_speed"`
	Condition   string  `json:"condition"`
	Unit        string  `json:"unit"`
}

// wttrResponse represents the JSON response from wttr.in API
type wttrResponse struct {
	CurrentCondition []wttrCurrentCondition `json:"current_condition"`
	NearestArea      []wttrNearestArea      `json:"nearest_area"`
}

type wttrCurrentCondition struct {
	TempC         string           `json:"temp_C"`
	TempF         string           `json:"temp_F"`
	FeelsLikeC    string           `json:"FeelsLikeC"`
	Humidity      string           `json:"humidity"`
	WindspeedKmph string           `json:"windspeedKmph"`
	WeatherDesc   []wttrValueField `json:"weatherDesc"`
}

type wttrNearestArea struct {
	AreaName []wttrValueField `json:"areaName"`
	Country  []wttrValueField `json:"country"`
}

type wttrValueField struct {
	Value string `json:"value"`
}

// weatherToolHandler implements the weather tool functionality
func weatherToolHandler(invocation copilot.ToolInvocation) (copilot.ToolResult, error) {
	// Parse parameters
	var params WeatherParams
	if err := mapToStruct(invocation.Arguments, &params); err != nil {
		return copilot.ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	// Fetch real weather data from wttr.in
	result, err := fetchWeatherFromAPI(params.City)
	if err != nil {
		return copilot.ToolResult{
			TextResultForLLM: fmt.Sprintf("Failed to get weather: %v", err),
			ResultType:       "error",
			SessionLog:       fmt.Sprintf("Weather API error: %v", err),
		}, nil
	}

	// Format the result for the LLM with more details
	locationStr := result.City
	if result.Country != "" {
		locationStr = fmt.Sprintf("%s, %s", result.City, result.Country)
	}

	textResult := fmt.Sprintf("Weather in %s: %.1f°C (feels like %.1f°C), %s. Humidity: %s, Wind: %s",
		locationStr, result.Temperature, result.FeelsLike, result.Condition, result.Humidity, result.WindSpeed)

	logMsg := fmt.Sprintf("Retrieved weather for %s", locationStr)
	if params.City == "" {
		logMsg = fmt.Sprintf("Retrieved weather for %s (auto-detected)", locationStr)
	}

	return copilot.ToolResult{
		TextResultForLLM: textResult,
		ResultType:       "success",
		SessionLog:       logMsg,
	}, nil
}

// fetchWeatherFromAPI fetches real weather data from wttr.in
func fetchWeatherFromAPI(city string) (WeatherResult, error) {
	// Build the URL - empty city means auto-detect by IP
	url := "https://wttr.in/"
	if city != "" {
		url += city
	}
	url += "?format=j1"

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make the request
	resp, err := client.Get(url)
	if err != nil {
		return WeatherResult{}, fmt.Errorf("failed to fetch weather: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return WeatherResult{}, fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherResult{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var wttr wttrResponse
	if err := json.Unmarshal(body, &wttr); err != nil {
		return WeatherResult{}, fmt.Errorf("failed to parse weather data: %w", err)
	}

	// Extract data from response
	if len(wttr.CurrentCondition) == 0 {
		return WeatherResult{}, fmt.Errorf("no weather data available")
	}

	current := wttr.CurrentCondition[0]

	// Get location info
	locationCity := city
	country := ""
	if len(wttr.NearestArea) > 0 {
		area := wttr.NearestArea[0]
		if len(area.AreaName) > 0 {
			locationCity = area.AreaName[0].Value
		}
		if len(area.Country) > 0 {
			country = area.Country[0].Value
		}
	}

	// Parse temperature
	var temp float64
	fmt.Sscanf(current.TempC, "%f", &temp)

	var feelsLike float64
	fmt.Sscanf(current.FeelsLikeC, "%f", &feelsLike)

	// Get weather description
	condition := "Unknown"
	if len(current.WeatherDesc) > 0 {
		condition = current.WeatherDesc[0].Value
	}

	return WeatherResult{
		City:        locationCity,
		Country:     country,
		Temperature: temp,
		FeelsLike:   feelsLike,
		Humidity:    current.Humidity + "%",
		WindSpeed:   current.WindspeedKmph + " km/h",
		Condition:   condition,
		Unit:        "Celsius",
	}, nil
}

// mapToStruct converts a map to a struct using JSON marshaling
func mapToStruct(data interface{}, target interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal to target: %w", err)
	}
	return nil
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
