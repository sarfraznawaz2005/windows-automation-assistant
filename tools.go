package main

import (
	"windows-assistant/usertools"

	copilot "github.com/github/copilot-sdk/go"
)

// loadCustomTools loads all registered tools from the usertools package
func loadCustomTools(config *Config) ([]copilot.Tool, error) {
	if !config.Tools.Enabled {
		return nil, nil
	}

	// Get all registered tools
	allTools := usertools.GetAll()

	// If no specific tools are listed in config, return all tools
	if len(config.Tools.EnabledTools) == 0 {
		return allTools, nil
	}

	// Filter to only enabled tools
	var enabledTools []copilot.Tool
	for _, tool := range allTools {
		if isToolEnabled(tool.Name, config) {
			enabledTools = append(enabledTools, tool)
		}
	}

	return enabledTools, nil
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
