# AGENTS.md - Windows Automation Assistant

This document provides guidelines and commands for agentic coding assistants working on the `Windows Automation Assistant` project.

NOTE: This project uses CopilotSdk: https://github.com/github/copilot-sdk

## MANDATORY: Build and Test After Every Change

**AI agents MUST always build and run tests after making any code changes to ensure the project remains in a working state.**

### Required Workflow

After making ANY code changes (editing, adding, or deleting code), execute:

```bash
# ALWAYS run this after making changes
go build -ldflags="-s -w" -o assistant.exe *.go && go test -v
```

### Verification Checklist

Before considering a task complete, verify:
1. **Build succeeds**: `go build -ldflags="-s -w" -o assistant.exe *.go` exits with code 0
2. **All tests pass**: `go test -v` shows all tests PASS
3. **No regressions**: Existing functionality still works

### If Build or Tests Fail

1. **DO NOT** move on to other tasks
2. **FIX** the failing build/tests immediately
3. **RE-RUN** the build and tests until they pass
4. Only then proceed with additional changes

### Quick Command Reference

```bash
# Build only
go build -ldflags="-s -w" -o assistant.exe *.go

# Test only
go test -v

# Build AND test (preferred - use this!)
go build -ldflags="-s -w" -o assistant.exe *.go && go test -v

# Test with coverage
go test -v -cover
```

### Build Flags

The project uses `-ldflags="-s -w"` for production builds:
- `-s` - Strips symbol table
- `-w` - Strips DWARF debug information

**Benefits:** Smaller binary (~30% smaller, ~6.9 MB vs ~9.8 MB), faster startup
**Trade-offs:** No line numbers in panic stack traces, debugger/profiler support limited

For development builds with full debugging support, omit these flags:
```bash
go build -o assistant_dev.exe *.go
```

---

## Build, Lint, and Test Commands

### Building the Assistant

```bash
# Build assistant (Windows)
go build -ldflags="-s -w" -o assistant.exe *.go

# Build with verbose output
go build -v -ldflags="-s -w" -o assistant.exe *.go
```

### Running Tests

```bash
# Run all tests (47 tests)
go test -v

# Run tests with coverage (~32% statement coverage)
go test -v -cover

# Run tests with race detection
go test -v -race

# Run a specific test
go test -v -run TestConfigValidation

# Run tests in a specific package (if subpackages exist)
go test -v ./...

# Run tests with verbose coverage profile
go test -v -coverprofile=coverage.out
go tool cover -func=coverage.out
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

## File Structure

```
windows-automation-assistant/
├── main.go              # Entry point (~31 lines)
├── cli.go               # CLI flags, usage, argument parsing (~133 lines)
├── config.go            # Configuration management (~200 lines)
├── errors.go            # Error handling utilities (~63 lines)
├── output.go            # Colors, JSON response, output helpers (~45 lines)
├── interactive.go       # Interactive mode conversation loop (~318 lines)
├── session.go           # Single command session execution (~205 lines)
 ├── progress.go          # Progress/spinner indicators (custom implementation)
 ├── markdown.go          # Markdown rendering (custom lightweight renderer)
├── tools.go             # Tool loading from usertools package (~50 lines)
├── assistant_test.go    # Comprehensive unit tests (~38 tests)
├── config.yaml          # Default configuration (auto-created)
├── AGENTS.md            # This file - guidelines for AI agents
├── README.md            # Project documentation
└── usertools/           # Custom tools package
    ├── registry.go      # Tool registry and helper functions
    ├── weather.go       # Weather tool (wttr.in API)
    └── sum.go           # Sum tool (example)
```

### File Responsibilities

| File | Purpose |
|------|---------|
| `main.go` | Minimal entry point - parses flags and dispatches to appropriate mode |
| `cli.go` | All CLI flag definitions, usage text, flag parsing logic |
| `config.go` | Config struct, loading, saving, validation, defaults |
| `errors.go` | Error handling, user-friendly error messages, debug output |
| `output.go` | ANSI colors, JSON response struct, terminal output helpers |
| `interactive.go` | Multi-turn conversation loop, special commands (help, config, clear), signal handling |
| `session.go` | Single-shot prompt execution with streaming support, signal handling |
| `progress.go` | Spinner/progress indicator (custom implementation) |
| `markdown.go` | Markdown rendering (custom lightweight renderer) |
| `tools.go` | Tool loading from usertools package, filtering by config |
| `usertools/registry.go` | Tool registry, registration, helper functions |
| `usertools/weather.go` | Weather tool implementation (wttr.in API) |
| `usertools/sum.go` | Sum tool implementation (example) |
| `assistant_test.go` | Comprehensive unit tests for all modules |

## Test Coverage Summary

The project has **38 tests**.

### Not Tested (Integration/Runtime):
- `main`, `runSingleCommand`, `runInteractiveMode` - Require Copilot SDK client
- `parseFlags`, `handleError`, `outputJSON` - CLI entry points / os.Exit calls

## CLI Usage

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

# Streaming options
assistant.exe --stream "real-time response"    # Force enable streaming (default)
assistant.exe --no-stream "wait for full"      # Disable streaming

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

## Code Style Guidelines

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
    "os/signal"
    "runtime"
    "strings"
    "sync"
    "syscall"

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
    noStream       = flag.Bool("no-stream", false, "Disable response streaming")
    stream         = flag.Bool("stream", false, "Force enable response streaming")
    configPath     = flag.String("config", "", "Path to config file")
    generateConfig = flag.Bool("generate-config", false, "Generate default config file and exit")
)
```

### Testing Guidelines

#### Test File Organization
- Tests go in `assistant_test.go` (single test file for this project)
- Test functions start with `Test`
- Use descriptive test names
- Group related tests with comments (e.g., `// ============ CONFIG.GO TESTS ============`)

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

## Quick Reference

**Build & Test:**
- `go build -o assistant.exe *.go` - Build the project
- `go test -v` - Run all tests (47 tests)
- `go test -v -cover` - Run tests with coverage (~32%)
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
- `assistant_test.go` - Add new tests here
