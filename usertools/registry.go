// Package usertools provides a registry for custom tools.
// Users can add their own tools by creating Go files in this package
// and registering them using the Register function in an init() block.
package usertools

import (
	"encoding/json"
	"fmt"

	copilot "github.com/github/copilot-sdk/go"
)

// Tool represents a complete tool definition with its handler
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
	Handler     copilot.ToolHandler
}

// registry holds all registered tools
var registry = make(map[string]Tool)

// Register adds a tool to the registry. Call this in init() of your tool file.
func Register(tool Tool) {
	if tool.Name == "" {
		panic("tool name cannot be empty")
	}
	if tool.Handler == nil {
		panic(fmt.Sprintf("tool %q handler cannot be nil", tool.Name))
	}
	registry[tool.Name] = tool
}

// GetAll returns all registered tools as copilot.Tool slice
func GetAll() []copilot.Tool {
	tools := make([]copilot.Tool, 0, len(registry))
	for _, t := range registry {
		tools = append(tools, copilot.Tool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
			Handler:     t.Handler,
		})
	}
	return tools
}

// Get returns a specific tool by name, or nil if not found
func Get(name string) *Tool {
	if t, ok := registry[name]; ok {
		return &t
	}
	return nil
}

// List returns the names of all registered tools
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered tools
func Count() int {
	return len(registry)
}

// MapToStruct converts a map to a struct using JSON marshaling.
// This is a helper function for tool handlers to parse arguments.
func MapToStruct(data interface{}, target interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal to target: %w", err)
	}
	return nil
}
