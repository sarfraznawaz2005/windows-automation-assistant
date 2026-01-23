# AGENTS.md - Windows Automation Assistant

This document provides guidelines and commands for agentic coding assistants working on the `Windows Automation Assistant` project.

NOTE: This project uses CopilotSdk: https://github.com/github/copilot-sdk

## 🚀 Build, Lint, and Test Commands

### Building the Assistant

```bash
# Navigate to assistant directory
cd assistant

# Build using the provided script
./build.sh

# Or build manually
go build -o assistant *.go

# Build with verbose output
go build -v -o assistant *.go
```

### Running Tests

```bash
# Run all tests
go test -v

# Run tests with coverage
go test -v -cover

# Run tests with race detection
go test -v -race

# Run a specific test
go test -v -run TestConfigValidation

# Run tests in a specific package (if subpackages exist)
go test -v ./...

# Run tests with verbose coverage profile
go test -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Linting and Code Quality

```bash
# Lint the code (from go directory)
cd go
golangci-lint run

# Lint with specific config
golangci-lint run --config .golangci.yml

# Fix auto-fixable issues
golangci-lint run --fix

# Lint specific file
golangci-lint run assistant.go
```

### Dependency Management

```bash
# Tidy dependencies
go mod tidy

# Download dependencies
go mod download

# Verify dependencies
go mod verify

# Clean module cache
go clean -modcache
```

## 📋 Code Style Guidelines

### General Principles

- **Consistency**: Follow existing patterns in the codebase
- **Readability**: Code should be self-documenting with clear variable names
- **Error Handling**: Use structured error handling with context
- **Testing**: Write tests for new functionality
- **Documentation**: Add comments for complex logic

### File Structure

```
assistant/
├── assistant.go          # Main application entry point
├── config.go            # Configuration management
├── tools.go             # Custom tools framework
├── progress.go          # Progress indicators
├── paths.go             # Windows path utilities
├── markdown.go          # Markdown rendering
├── assistant_test.go    # Unit tests
├── config.yaml          # Default configuration
├── build.sh             # Build script
└── README.md            # Documentation
```

### Go Formatting and Style

#### Imports
```go
import (
    // Standard library imports first (alphabetically sorted)
    "bufio"
    "errors"
    "flag"
    "fmt"
    "os"
    "runtime"
    "strings"

    // Third-party imports (alphabetically sorted)
    "github.com/charmbracelet/glamour"
    copilot "github.com/github/copilot-sdk/go"
    "gopkg.in/yaml.v3"
)
```

#### Package Declaration
- Use `package main` for CLI applications
- Use descriptive package names for libraries

#### Constants
```go
// Group related constants together
const (
    red    = "\033[31m"
    reset  = "\033[0m"
    yellow = "\033[33m"
)
```

#### Variable Naming
- **Local variables**: camelCase (`userInput`, `configPath`)
- **Global variables**: camelCase with clear purpose
- **Constants**: ALL_CAPS for global constants
- **Struct fields**: PascalCase for exported, camelCase for unexported

#### Function Naming
- **Exported functions**: PascalCase (`LoadConfig`, `ValidateConfig`)
- **Private functions**: camelCase (`handleError`, `getUserFriendlyError`)
- **Test functions**: PascalCase starting with `Test` (`TestConfigValidation`)

#### Struct Definitions
```go
// Use PascalCase for exported structs
type Config struct {
    Model        string       `yaml:"model" json:"model"`
    Debug        bool         `yaml:"debug" json:"debug"`
    SystemPrompt string       `yaml:"system_prompt" json:"system_prompt"`
    Output       OutputConfig `yaml:"output" json:"output"`
    Tools        ToolsConfig  `yaml:"tools" json:"tools"`
}

// Use camelCase for private structs
type errorInfo struct {
    message  string
    file     string
    line     int
    function string
}
```

#### Error Handling

```go
// Use custom error handling function
func handleError(err error, context string) {
    if err == nil {
        return
    }
    // Structured error handling with context
}

// Return errors with context
func LoadConfig(configPath string) (*Config, error) {
    if configPath == "" {
        return nil, fmt.Errorf("config path cannot be empty")
    }
    // ...
}

// Use error wrapping for context
return fmt.Errorf("failed to load config from %s: %w", configPath, err)
```

#### YAML Configuration
```go
// Use proper YAML tags
type Config struct {
    Model string `yaml:"model" json:"model"`
    Debug bool   `yaml:"debug" json:"debug"`
}

// Use pointer types for optional values
type ClientConfig struct {
    AutoRestart *bool `yaml:"auto_restart" json:"auto_restart"`
    AutoStart   *bool `yaml:"auto_start" json:"auto_start"`
}
```

#### CLI Flag Definitions
```go
// Group related flags
var (
    interactive    = flag.Bool("interactive", false, "Enable interactive mode")
    i              = flag.Bool("i", false, "Enable interactive mode (short)")
    jsonOutput     = flag.Bool("json", false, "Output in JSON format")
    configPath     = flag.String("config", "", "Path to config file")
    generateConfig = flag.Bool("generate-config", false, "Generate default config file and exit")
)
```

#### Comments and Documentation

```go
// Package-level documentation
// Package main provides the Windows Automation Assistant CLI application.

// Function documentation
// LoadConfig loads configuration from file or returns default
func LoadConfig(configPath string) (*Config, error) {
    // Implementation comments for complex logic
    if configPath == "" {
        // Try default locations
        configPath = findConfigFile()
    }
}
```

### Testing Guidelines

#### Test File Organization
- Tests go in `*_test.go` files in the same package
- Test functions start with `Test`
- Use descriptive test names

#### Test Structure
```go
func TestConfigValidation(t *testing.T) {
    // Test valid config
    config := DefaultConfig()
    if err := ValidateConfig(config); err != nil {
        t.Errorf("Default config should be valid: %v", err)
    }

    // Test invalid config
    config.Model = ""
    if err := ValidateConfig(config); err == nil {
        t.Error("Config with empty model should be invalid")
    }
}
```

#### Table-Driven Tests
```go
func TestPathNormalization(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"basic path", "test/path", "test/path"},
        {"absolute path", "/absolute/path", "/absolute/path"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := NormalizePath(tt.input)
            if result != tt.expected {
                t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
            }
        })
    }
}
```

### Code Organization

#### Function Length
- Keep functions focused and under 50 lines
- Extract complex logic into helper functions
- Use early returns to reduce nesting

#### File Size
- Keep files under 500 lines
- Split large files by functionality
- Use clear file naming conventions

#### Dependency Management
- Use Go modules for dependency management
- Keep dependencies minimal and well-maintained
- Update dependencies regularly
- Document major dependency changes

### Git and Version Control

#### Commit Messages
```
feat: add markdown rendering support
fix: handle empty config file gracefully
docs: update README with new features
test: add unit tests for path normalization
```

#### Branch Naming
- `feature/add-markdown-support`
- `fix/config-validation`
- `docs/update-readme`

### Performance Considerations

- Use efficient data structures
- Avoid unnecessary allocations
- Profile performance-critical code
- Use appropriate concurrency patterns

### Security Best Practices

- Validate all inputs
- Use safe YAML/JSON parsing
- Avoid command injection
- Sanitize file paths
- Handle sensitive data appropriately

---

## 🎯 Quick Reference

**Build & Test:**
- `./build.sh` - Build the project
- `go build -o assistant *.go` - Manual build
- `go test -v` - Run all tests
- `go test -v -run TestName` - Run specific test
- `cd ../go && golangci-lint run` - Lint the SDK code
- `./assistant --help` - Show help (Windows-compatible)

**Code Style:**
- Standard Go formatting with `gofmt`
- Imports: stdlib first, then third-party (alphabetical)
- Functions: PascalCase for exported, camelCase for private
- Errors: Use structured error handling with context
- Tests: Descriptive names, table-driven where appropriate

**File Structure:**
- One responsibility per file
- Clear naming conventions
- Comprehensive documentation
- Unit tests for all functionality</content>
<parameter name="filePath">D:\SystemFolders\Downloads\copilot-sdk-main\AGENTS.md