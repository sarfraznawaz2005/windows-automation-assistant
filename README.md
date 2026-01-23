# Windows Automation Assistant

A powerful AI agent built with [GitHub Copilot SDK](https://github.com/github/copilot-sdk) for Windows 11 automation tasks, featuring interactive mode, custom tools, and comprehensive configuration options.

## Features

- **Windows Automation**: Specialized for Windows 11 tasks using available tools
- **Interactive Mode**: Multi-turn conversations with `--interactive` or `-i` flag
- **Streaming Responses**: Real-time streaming with progress indicators
- **Markdown Rendering**: Beautiful terminal output with glamour
- **Custom Tools**: Extensible tool system with Go-based tool definitions
- **JSON Output**: Structured output for programmatic use with `--json` flag
- **YAML Configuration**: Customizable settings via `config.yaml`

## Installation

### Prerequisites

- **Go**: 1.21+
- **GitHub Copilot CLI**: Installed and authenticated
- **Windows**: 11 environment (primary target)

### Build

```bash
go build -o assistant.exe *.go
```

## Usage

### Single Command Mode
```bash
# Basic usage
assistant.exe "list files in current directory"

# With specific model
assistant.exe "show disk usage" gpt-4.1

# Weather tool example
assistant.exe "what's the weather like in Tokyo"
```

### Interactive Mode
```bash
assistant.exe --interactive
# or
assistant.exe -i
```

### Output Options
```bash
# JSON output for programmatic use
assistant.exe --json "analyze this file"

# Force markdown rendering (only works with --no-stream)
assistant.exe --markdown --no-stream "create a table"

# Disable markdown
assistant.exe --no-markdown "simple output"

# Control streaming
assistant.exe --stream "stream response in real-time"
assistant.exe --no-stream "wait for full response"

# Control spinner
assistant.exe --spinner "long task"
assistant.exe --no-spinner "quick task"
```

> **Note:** When `--stream` is enabled (default), `--markdown` has no effect since content is printed in real-time as it arrives. Use `--no-stream --markdown` to enable markdown rendering.

### Configuration
```bash
# Generate default config file
assistant.exe --generate-config

# Use custom config
assistant.exe --config /path/to/config.yaml "task"

# Show help
assistant.exe --help
```

## Configuration

The assistant looks for configuration in this order:
1. `./config.yaml` (current directory)
2. `~/.assistant-config.yaml` (home directory)

### Example config.yaml

```yaml
model: gpt-4.1
debug: false
system_prompt: |
  You are "Windows Automation Assistant"...
output:
  markdown: false
  json: false
  spinner: true
  streaming: true
tools:
  enabled: true
  enabled_tools: []  # Empty means all tools enabled, or specify: ["weather", "sum"]
client:
  log_level: error
  auto_restart: true
  auto_start: true
```

## Interactive Mode Commands

When in interactive mode:

| Command | Description |
|---------|-------------|
| `help`, `h`, `?` | Show available commands |
| `clear`, `cls` | Clear the screen |
| `config` | Show current configuration |
| `exit`, `quit`, `bye`, `q` | Exit interactive mode |

## JSON Output Format

When using `--json` flag:

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

## Custom Tools

Custom tools are Go files in the `usertools/` package. Each tool is self-contained with its definition and handler in a single file. Tools are automatically registered at startup via Go's `init()` mechanism.

### Built-in Tools

| Tool | Description |
|------|-------------|
| `weather` | Get current weather for a city (uses wttr.in API) |
| `sum` | Add two numbers together |

### Creating a New Tool

Follow these steps to add a custom tool:

#### Step 1: Create a New Go File

Create a new file in the `usertools/` directory (e.g., `usertools/mytool.go`):

```go
package usertools

import (
    "fmt"
    copilot "github.com/github/copilot-sdk/go"
)

func init() {
    Register(Tool{
        Name:        "mytool",
        Description: "Description of what your tool does",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "param1": map[string]interface{}{
                    "type":        "string",
                    "description": "Description of param1",
                },
                "param2": map[string]interface{}{
                    "type":        "number",
                    "description": "Description of param2",
                },
            },
            "required": []string{"param1"},
        },
        Handler: myToolHandler,
    })
}

// Define your parameters struct
type myToolParams struct {
    Param1 string  `json:"param1"`
    Param2 float64 `json:"param2"`
}

// Implement the handler function
func myToolHandler(invocation copilot.ToolInvocation) (copilot.ToolResult, error) {
    // Parse parameters
    var params myToolParams
    if err := MapToStruct(invocation.Arguments, &params); err != nil {
        return copilot.ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
    }

    // Your tool logic here
    result := fmt.Sprintf("Processed: %s with value %v", params.Param1, params.Param2)

    return copilot.ToolResult{
        TextResultForLLM: result,
        ResultType:       "success",
        SessionLog:       "Tool executed successfully",
    }, nil
}
```

#### Step 2: Rebuild the Project

```bash
go build -o assistant.exe .
```

#### Step 3: Test Your Tool

```bash
assistant.exe "use mytool with param1=hello and param2=42"
```

### Complete Example: Sum Tool

Here's a complete example of the `sum` tool (`usertools/sum.go`):

```go
package usertools

import (
    "fmt"
    copilot "github.com/github/copilot-sdk/go"
)

func init() {
    Register(Tool{
        Name:        "sum",
        Description: "Adds two numbers together and returns the result",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "a": map[string]interface{}{
                    "type":        "number",
                    "description": "The first number",
                },
                "b": map[string]interface{}{
                    "type":        "number",
                    "description": "The second number",
                },
            },
            "required": []string{"a", "b"},
        },
        Handler: sumHandler,
    })
}

type sumParams struct {
    A float64 `json:"a"`
    B float64 `json:"b"`
}

func sumHandler(invocation copilot.ToolInvocation) (copilot.ToolResult, error) {
    var params sumParams
    if err := MapToStruct(invocation.Arguments, &params); err != nil {
        return copilot.ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
    }

    result := params.A + params.B
    textResult := fmt.Sprintf("The sum of %v and %v is %v", params.A, params.B, result)

    return copilot.ToolResult{
        TextResultForLLM: textResult,
        ResultType:       "success",
        SessionLog:       fmt.Sprintf("Calculated: %v + %v = %v", params.A, params.B, result),
    }, nil
}
```

### Tool Structure Reference

| Field | Type | Description |
|-------|------|-------------|
| `Name` | string | Unique tool identifier (used by the LLM to call the tool) |
| `Description` | string | What the tool does (helps the LLM decide when to use it) |
| `Parameters` | map | JSON Schema defining the tool's input parameters |
| `Handler` | function | The function that executes when the tool is called |

### ToolResult Fields

| Field | Type | Description |
|-------|------|-------------|
| `TextResultForLLM` | string | The result text returned to the LLM |
| `ResultType` | string | "success" or "error" |
| `SessionLog` | string | Log message for debugging |

### Helper Functions

The `usertools` package provides helper functions:

- `MapToStruct(data, target)` - Converts map arguments to a typed struct
- `Register(tool)` - Registers a tool (call in `init()`)
- `Get(name)` - Get a tool by name
- `GetAll()` - Get all registered tools
- `List()` - Get names of all registered tools
- `Count()` - Get number of registered tools

## Testing

Run the comprehensive test suite:

```bash
# Run all tests
go test -v

# Run with coverage
go test -v -cover

# Run specific test
go test -v -run TestConfigValidation
```

## Dependencies

- [github.com/github/copilot-sdk/go](https://github.com/github/copilot-sdk) - GitHub Copilot SDK
- [github.com/briandowns/spinner](https://github.com/briandowns/spinner) - Terminal spinner
- [github.com/charmbracelet/glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing

## License

MIT License
