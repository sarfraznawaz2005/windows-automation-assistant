package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// ANSI color codes for terminal output
const (
	colorRed    = "\033[31m"
	colorReset  = "\033[0m"
	colorYellow = "\033[33m"
)

// JSONResponse is the structure for JSON output mode
type JSONResponse struct {
	Success  bool     `json:"success"`
	Response string   `json:"response,omitempty"`
	Error    string   `json:"error,omitempty"`
	Model    string   `json:"model,omitempty"`
	Tools    []string `json:"tools_used,omitempty"`
}

// safeColor returns color code (ANSI is widely supported on modern terminals)
func safeColor(color string) string {
	return color
}

// outputJSON outputs a JSON response to stdout
func outputJSON(success bool, response, errMsg, model string, toolsUsed []string) {
	jsonResp := JSONResponse{
		Success:  success,
		Response: response,
		Error:    errMsg,
		Model:    model,
		Tools:    toolsUsed,
	}
	jsonBytes, err := json.Marshal(jsonResp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to marshal JSON response: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonBytes))
}
