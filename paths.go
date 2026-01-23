package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NormalizePath normalizes a path for cross-platform compatibility
func NormalizePath(path string) string {
	// Expand environment variables
	path = os.ExpandEnv(path)

	// Convert to absolute path if relative
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err == nil {
			path = absPath
		}
	}

	// Clean the path (removes redundant separators, etc.)
	path = filepath.Clean(path)

	// Handle Windows-specific path issues
	if isWindows() {
		path = normalizeWindowsPath(path)
	}

	return path
}

// normalizeWindowsPath handles Windows-specific path normalization
func normalizeWindowsPath(path string) string {
	// Convert forward slashes to backslashes for consistency
	path = strings.ReplaceAll(path, "/", "\\")

	// Handle UNC paths (\\server\share)
	if strings.HasPrefix(path, "\\\\") {
		return path
	}

	// Handle long path prefix (\\?\)
	if strings.HasPrefix(path, "\\\\?\\") {
		return path
	}

	// Add long path prefix if path is longer than MAX_PATH (260 chars)
	// This allows Windows to handle paths longer than 260 characters
	if len(path) > 260 && !strings.HasPrefix(path, "\\\\?\\") {
		// Only add prefix for absolute paths
		if filepath.IsAbs(path) {
			return "\\\\?\\" + path
		}
	}

	return path
}

// ExpandPath expands a path with environment variables and handles special cases
func ExpandPath(path string) string {
	// Handle common Windows environment variables
	replacements := map[string]string{
		"%USERPROFILE%":       os.Getenv("USERPROFILE"),
		"%APPDATA%":           os.Getenv("APPDATA"),
		"%LOCALAPPDATA%":      os.Getenv("LOCALAPPDATA"),
		"%PROGRAMFILES%":      os.Getenv("PROGRAMFILES"),
		"%PROGRAMFILES(X86)%": os.Getenv("PROGRAMFILES(X86)"),
		"%WINDIR%":            os.Getenv("WINDIR"),
		"%SYSTEMROOT%":        os.Getenv("SYSTEMROOT"),
		"%TEMP%":              os.Getenv("TEMP"),
		"%TMP%":               os.Getenv("TMP"),
	}

	for placeholder, value := range replacements {
		path = strings.ReplaceAll(path, placeholder, value)
	}

	// Also handle lowercase versions
	for placeholder, value := range replacements {
		path = strings.ReplaceAll(path, strings.ToLower(placeholder), value)
	}

	// Use standard os.ExpandEnv for any remaining variables
	path = os.ExpandEnv(path)

	return path
}

// ValidatePath checks if a path exists and is accessible
func ValidatePath(path string) error {
	path = NormalizePath(path)

	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", path)
	} else if err != nil {
		return fmt.Errorf("cannot access path: %s (%v)", path, err)
	}

	return nil
}

// GetSafePath returns a safe version of the path for display/logging
func GetSafePath(path string) string {
	path = NormalizePath(path)

	// On Windows, we might want to show a shorter version for display
	if isWindows() && len(path) > 80 {
		// Show ... in the middle for very long paths
		if len(path) > 120 {
			return path[:40] + "..." + path[len(path)-40:]
		}
	}

	return path
}

// isWindows checks if running on Windows
func isWindows() bool {
	return os.Getenv("OS") == "Windows_NT" || strings.Contains(strings.ToLower(os.Getenv("OS")), "windows")
}

// ResolvePath resolves a path with various strategies
func ResolvePath(path string) (string, error) {
	// Try the path as-is first
	if err := ValidatePath(path); err == nil {
		return NormalizePath(path), nil
	}

	// Try expanding environment variables
	expanded := ExpandPath(path)
	if err := ValidatePath(expanded); err == nil {
		return NormalizePath(expanded), nil
	}

	// Try relative to current directory
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			fullPath := filepath.Join(cwd, path)
			if err := ValidatePath(fullPath); err == nil {
				return NormalizePath(fullPath), nil
			}
		}
	}

	// Try common Windows locations
	commonLocations := []string{
		os.Getenv("USERPROFILE"),
		os.Getenv("PROGRAMFILES"),
		os.Getenv("PROGRAMFILES(X86)"),
		os.Getenv("WINDIR"),
		"C:\\",
		"C:\\Windows",
		"C:\\Program Files",
	}

	for _, base := range commonLocations {
		if base != "" {
			fullPath := filepath.Join(base, path)
			if err := ValidatePath(fullPath); err == nil {
				return NormalizePath(fullPath), nil
			}
		}
	}

	return "", fmt.Errorf("could not resolve path: %s", path)
}
