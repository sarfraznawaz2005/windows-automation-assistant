// Weather tool - fetches real weather data from wttr.in API
// This tool demonstrates lazy loading - the HTTP client and handler
// are only initialized when the tool is first called.
package usertools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func init() {
	RegisterLazy(ToolDefinition{
		Name:        "weather",
		Description: "Get current weather for a city. If no city is provided, auto-detects location based on IP.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"city": map[string]interface{}{
					"type":        "string",
					"description": "The city name (optional - will auto-detect if not provided)",
				},
			},
		},
		Loader: loadWeatherHandler,
	})
}

// Shared HTTP client - only created when handler is first loaded
var weatherHTTPClient *http.Client

// loadWeatherHandler initializes the weather tool and returns the handler
func loadWeatherHandler() ToolHandler {
	// Initialize HTTP client with connection pooling
	weatherHTTPClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	return weatherHandler
}

// weatherParams defines the parameters for the weather tool
type weatherParams struct {
	City string `json:"city"`
}

// weatherResult defines the result structure
type weatherResult struct {
	City        string  `json:"city"`
	Country     string  `json:"country"`
	Temperature float64 `json:"temperature"`
	FeelsLike   float64 `json:"feels_like"`
	Humidity    string  `json:"humidity"`
	WindSpeed   string  `json:"wind_speed"`
	Condition   string  `json:"condition"`
	Unit        string  `json:"unit"`
}

// wttr.in API response structures
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

// weatherHandler implements the weather tool functionality
func weatherHandler(invocation ToolInvocation) (ToolResult, error) {
	// Parse parameters
	var params weatherParams
	if err := MapToStruct(invocation.Arguments, &params); err != nil {
		return ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	// Fetch real weather data from wttr.in
	result, err := fetchWeather(params.City)
	if err != nil {
		return ToolResult{
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

	return ToolResult{
		TextResultForLLM: textResult,
		ResultType:       "success",
		SessionLog:       logMsg,
	}, nil
}

// fetchWeather fetches real weather data from wttr.in
func fetchWeather(city string) (weatherResult, error) {
	// Build the URL - empty city means auto-detect by IP
	url := "https://wttr.in/"
	if city != "" {
		url += city
	}
	url += "?format=j1"

	// Make the request using the lazily-initialized HTTP client
	resp, err := weatherHTTPClient.Get(url)
	if err != nil {
		return weatherResult{}, fmt.Errorf("failed to fetch weather: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return weatherResult{}, fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return weatherResult{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var wttr wttrResponse
	if err := json.Unmarshal(body, &wttr); err != nil {
		return weatherResult{}, fmt.Errorf("failed to parse weather data: %w", err)
	}

	// Extract data from response
	if len(wttr.CurrentCondition) == 0 {
		return weatherResult{}, fmt.Errorf("no weather data available")
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

	return weatherResult{
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
