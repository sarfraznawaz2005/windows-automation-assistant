package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
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

// supportsANSI checks if the terminal supports ANSI colors
func supportsANSI() bool {
	// On Windows, check if we're in Windows Terminal or similar
	if runtime.GOOS == "windows" {
		term := os.Getenv("TERM")
		wtSession := os.Getenv("WT_SESSION") // Windows Terminal
		if term == "xterm-256color" || wtSession != "" {
			return true
		}
		// For cmd.exe, ANSI might not work, but let's try anyway
		return true
	}
	return true // Assume ANSI support on Unix-like systems
}

// safeColor returns color code if supported, otherwise empty string
func safeColor(color string) string {
	if supportsANSI() {
		return color
	}
	return ""
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
	jsonBytes, _ := json.Marshal(jsonResp)
	fmt.Println(string(jsonBytes))
}
