# Windows Automation Assistant

A powerful AI agent built with [GitHub Copilot SDK](https://github.com/github/copilot-sdk) for Windows 11 automation tasks, featuring interactive mode, custom tools, and comprehensive configuration options.

## Features

- **Windows Automation**: Specialized for Windows 11 tasks using available tools
- **Interactive Mode**: Multi-turn conversations with `--interactive` or `-i` flag
- **Streaming Responses**: Real-time streaming with progress indicators
- **Markdown Rendering**: Beautiful terminal output with glamour
- **Custom Tools**: Extensible tool system with YAML-based definitions
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

# Force markdown rendering
assistant.exe --markdown "create a table"

# Disable markdown
assistant.exe --no-markdown "simple output"

# Control streaming
assistant.exe --stream "stream response in real-time"
assistant.exe --no-stream "wait for full response"

# Control spinner
assistant.exe --spinner "long task"
assistant.exe --no-spinner "quick task"
```

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
  directory: user-tools
  enabled_tools:
    - weather
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

Create custom tools by adding YAML files to the `user-tools/` directory:

```yaml
# user-tools/weather.yaml
name: weather
description: Get current weather for a city
handler: weather
parameters:
  type: object
  properties:
    city:
      type: string
      description: The city name
  required:
    - city
```

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
