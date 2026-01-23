package main

import (
	"fmt"
	"os"
)

func main() {
	// Parse CLI flags and load configuration
	config := parseFlags()

	// Determine mode
	if isInteractiveMode() {
		// Debug: show which model is being used
		if config.Debug {
			fmt.Fprintf(os.Stderr, "%s[DEBUG] Using model: %s%s\n", safeColor(colorYellow), config.Model, safeColor(colorReset))
		}
		runInteractiveMode(config)
		return
	}

	// Single command mode
	prompt, model := getPromptAndModel(config)

	// Debug: show which model is being used
	if config.Debug {
		fmt.Fprintf(os.Stderr, "%s[DEBUG] Using model: %s%s\n", safeColor(colorYellow), model, safeColor(colorReset))
	}

	runSingleCommand(config, prompt, model)
}
