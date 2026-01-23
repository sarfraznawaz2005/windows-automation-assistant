# AGENTS.md - Windows Automation Assistant

This document provides guidelines and commands for agentic coding assistants working on the `Windows Automation Assistant` project.

NOTE: This project uses CopilotSdk: https://github.com/github/copilot-sdk

## 🚀 Build, Lint, and Test Commands

### Building the Assistant

```bash
# Build the assistant (Windows)
go build -o assistant.exe *.go

# Build with verbose output
go build -v -o assistant.exe *.go
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
# Lint the code
golangci-lint run

# Lint with specific config
golangci-lint run --config .golangci.yml

# Fix auto-fixable issues
golangci-lint run --fix

# Lint specific file
golangci-lint run main.go
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

## 📁 File Structure

```
windows-automation-assistant/
├── main.go              # Entry point (~30 lines)
├── cli.go               # CLI flags, usage, argument parsing (~130 lines)
├── config.go            # Configuration management (~214 lines)
├── errors.go            # Error handling utilities (~80 lines)
├── output.go            # Colors, JSON response, output helpers (~60 lines)
├── interactive.go       # Interactive mode conversation loop (~280 lines)
├── session.go           # Single command session execution (~175 lines)
├── progress.go          # Progress/spinner indicators (~57 lines)
├── markdown.go          # Markdown rendering with glamour (~25 lines)
├── tools.go             # Custom tools framework (~214 lines)
├── assistant_test.go    # Unit tests
├── config.yaml          # Default configuration (auto-created)
└── user-tools/          # Directory for custom tool definitions
    └── weather.yaml     # Example weather tool
```

### File Responsibilities

| File | Purpose |
|------|---------|
| `main.go` | Minimal entry point - parses flags and dispatches to appropriate mode |
| `cli.go` | All CLI flag definitions, usage text, flag parsing logic |
| `config.go` | Config struct, loading, saving, validation, defaults |
| `errors.go` | Error handling, user-friendly error messages, debug output |
| `output.go` | ANSI colors, JSON response struct, terminal output helpers |
| `interactive.go` | Multi-turn conversation loop, special commands (help, config, clear) |
| `session.go` | Single-shot prompt execution with streaming support |
| `progress.go` | Spinner/progress indicator using briandowns/spinner |
| `markdown.go` | Markdown rendering using charmbracelet/glamour |
| `tools.go` | Custom tool loading from YAML, tool handler registry |

## 🎮 CLI Usage

```bash
# Single command mode
assistant.exe "list files in current directory"
assistant.exe "analyze this file" gpt-4.1

# Interactive mode
assistant.exe -i
assistant.exe --interactive

# Output options
assistant.exe --json "who are you?"           # JSON output for programmatic use
assistant.exe --markdown "create a table"      # Force markdown rendering
assistant.exe --no-markdown "simple output"    # Disable markdown

# Spinner options
assistant.exe --spinner "long task"            # Force enable spinner
assistant.exe --no-spinner "quick task"        # Disable spinner

# Configuration
assistant.exe --config /path/to/config.yaml "prompt"
assistant.exe --generate-config               # Create default config.yaml

# Help
assistant.exe --help
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ASSISTANT_DEBUG=1` | Show detailed error info with file/line numbers |
| `NO_SPINNER=1` | Disable progress spinner animations |

### JSON Output Format

When using `--json` flag, output is structured JSON for programmatic consumption:

```json
{
  "success": true,
  "response": "The assistant's response text...",
  "model": "gpt-4.1",
  "tools_used": ["weather", "shell"]
}
```

On error:
```json
{
  "success": false,
  "error": "Error message here",
  "model": "gpt-4.1",
  "tools_used": []
}
```

### Interactive Mode Commands

| Command | Description |
|---------|-------------|
| `help`, `h`, `?` | Show available commands |
| `clear`, `cls` | Clear the screen |
| `config` | Show current configuration |
| `exit`, `quit`, `bye`, `q` | Exit interactive mode |

## 📋 Code Style Guidelines

### General Principles

- **Consistency**: Follow existing patterns in the codebase
- **Readability**: Code should be self-documenting with clear variable names
- **Error Handling**: Use structured error handling with context
- **Testing**: Write tests for new functionality
- **Documentation**: Add comments for complex logic
- **Modularity**: One responsibility per file, keep files under 300 lines

### Go Formatting and Style

#### Imports
```go
import (
    // Standard library imports first (alphabetically sorted)
    "bufio"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "os"
    "runtime"
    "strings"
    "sync"

    // Third-party imports (alphabetically sorted)
    "github.com/briandowns/spinner"
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
// Group related constants together in output.go
const (
    colorRed    = "\033[31m"
    colorReset  = "\033[0m"
    colorYellow = "\033[33m"
)
```

#### Variable Naming
- **Local variables**: camelCase (`userInput`, `configPath`)
- **Global variables**: camelCase with clear purpose
- **Constants**: camelCase for package-level (colorRed, colorYellow)
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

// JSONResponse for --json output mode (in output.go)
type JSONResponse struct {
    Success  bool     `json:"success"`
    Response string   `json:"response,omitempty"`
    Error    string   `json:"error,omitempty"`
    Model    string   `json:"model,omitempty"`
    Tools    []string `json:"tools_used,omitempty"`
}
```

#### Error Handling

```go
// Use custom error handling function (in errors.go)
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

#### CLI Flag Definitions (in cli.go)
```go
// Group related flags
var (
    interactive    = flag.Bool("interactive", false, "Enable interactive mode")
    i              = flag.Bool("i", false, "Enable interactive mode (short)")
    jsonOutput     = flag.Bool("json", false, "Output in JSON format")
    noMarkdown     = flag.Bool("no-markdown", false, "Disable markdown rendering")
    markdown       = flag.Bool("markdown", false, "Force enable markdown rendering")
    noSpinner      = flag.Bool("no-spinner", false, "Disable loading spinner")
    showSpinner    = flag.Bool("spinner", false, "Force enable loading spinner")
    configPath     = flag.String("config", "", "Path to config file")
    generateConfig = flag.Bool("generate-config", false, "Generate default config file and exit")
)
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
func TestErrorMessages(t *testing.T) {
    tests := []struct {
        name     string
        err      error
        contains string
    }{
        {"connection error", errors.New("connection refused"), "Cannot connect"},
        {"auth error", errors.New("unauthorized"), "Authentication failed"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := getUserFriendlyError(tt.err, "test")
            if !strings.Contains(result, tt.contains) {
                t.Errorf("Expected error containing %q, got %q", tt.contains, result)
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
- Keep files under 300 lines
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
feat: add JSON output mode for programmatic use
fix: handle empty config file gracefully
refactor: modularize assistant.go into separate files
docs: update AGENTS.md with new file structure
test: add unit tests for error handling
```

#### Branch Naming
- `feature/add-json-output`
- `fix/config-validation`
- `refactor/modularize-code`
- `docs/update-agents-md`

### Performance Considerations

- Use efficient data structures
- Avoid unnecessary allocations
- Profile performance-critical code
- Use appropriate concurrency patterns (sync.Mutex for shared state in interactive mode)

### Security Best Practices

- Validate all inputs
- Use safe YAML/JSON parsing
- Avoid command injection
- Sanitize file paths
- Handle sensitive data appropriately

---

## 🎯 Quick Reference

**Build & Test:**
- `go build -o assistant.exe *.go` - Build the project
- `go test -v` - Run all tests
- `go test -v -run TestName` - Run specific test
- `golangci-lint run` - Lint the code
- `assistant.exe --help` - Show help

**Code Style:**
- Standard Go formatting with `gofmt`
- Imports: stdlib first, then third-party (alphabetical)
- Functions: PascalCase for exported, camelCase for private
- Errors: Use structured error handling with context
- Tests: Descriptive names, table-driven where appropriate

**File Organization:**
- One responsibility per file
- Keep files under 300 lines
- Clear naming conventions
- Comprehensive documentation
- Unit tests for all functionality

**Key Files to Know:**
- `cli.go` - Add new CLI flags here
- `config.go` - Add new config options here
- `errors.go` - Add new error handling here
- `output.go` - Add new output formats here
- `interactive.go` - Modify interactive mode behavior
- `session.go` - Modify single command behavior
- `tools.go` - Add new custom tool handlers
