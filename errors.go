package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// handleError gracefully handles errors with user-friendly output
func handleError(err error, context string) {
	if err == nil {
		return
	}

	// Show user-friendly error message
	fmt.Fprintf(os.Stderr, "%sError: %s%s\n", safeColor(colorRed), getUserFriendlyError(err, context), safeColor(colorReset))

	// Show detailed error info for debugging (only in verbose mode or for developers)
	if os.Getenv("ASSISTANT_DEBUG") == "1" {
		pc, file, line, ok := runtime.Caller(1)
		funcName := "unknown"
		if ok {
			funcName = runtime.FuncForPC(pc).Name()
			// Extract just the function name
			if lastSlash := strings.LastIndex(funcName, "/"); lastSlash >= 0 {
				funcName = funcName[lastSlash+1:]
			}
			if lastDot := strings.LastIndex(funcName, "."); lastDot >= 0 {
				funcName = funcName[lastDot+1:]
			}
		}
		fmt.Fprintf(os.Stderr, "%s[DEBUG] %s:%d in %s: %s%s\n",
			safeColor(colorYellow), file, line, funcName, err.Error(), safeColor(colorReset))
	}

	os.Exit(1)
}

// getUserFriendlyError converts technical errors to user-friendly messages
func getUserFriendlyError(err error, context string) string {
	errMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "dial tcp"):
		return "Cannot connect to GitHub Copilot CLI. Please ensure Copilot CLI is installed and running."
	case strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "unauthorized"):
		return "Authentication failed. Please run 'gh auth login' to authenticate with GitHub."
	case strings.Contains(errMsg, "model") && strings.Contains(errMsg, "not found"):
		return "The specified model is not available. Please check the model name or use the default."
	case strings.Contains(errMsg, "timeout"):
		return "Request timed out. Please try again."
	case strings.Contains(errMsg, "rate limit"):
		return "Rate limit exceeded. Please wait and try again."
	case strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "access denied"):
		return "Permission denied. Please check file permissions or authentication."
	default:
		if context != "" {
			return fmt.Sprintf("%s failed: %s", context, err.Error())
		}
		return err.Error()
	}
}
