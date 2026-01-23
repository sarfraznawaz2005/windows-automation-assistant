# Windows Automation Assistant

A powerful AI agent built with [GitHub Copilot SDK](https://github.com/github/copilot-sdk) for Windows 11 automation tasks, featuring interactive mode, custom tools, and comprehensive configuration options.

## 🚀 Features

- **Windows Automation**: Specialized for Windows 11 tasks using available tools
- **File System Access**: Full access to file operations (read, write, rename, etc.)
- **Interactive Mode**: Multi-turn conversations with `--interactive` or `-i` flag
- **YAML Configuration**: Customizable settings via `config.yaml`
- **Custom Tools**: Extensible tool system with weather tool example
- **JSON Output**: Structured output for programmatic use with `--json` flag

## 📋 Usage

### Single Command Mode
```bash
./assistant "your automation task" [model]
```

### Interactive Mode
```bash
./assistant --interactive
# or
./assistant -i
```

### JSON Output Mode
```bash
./assistant --json "task description"
```

### Generate Config
```bash
./assistant --generate-config
```

### Custom Config
```bash
./assistant --config path/to/config.yaml "task"
```

## ⚙️ Configuration

The assistant uses YAML configuration file `config.yaml`:

```yaml
model: gpt-4.1
debug: false
system_prompt: |
  Custom system prompt...
output:
  markdown: true
  json: false
tools:
  enabled: true
  directory: user-tools
  enabled_tools:
    - weather
```

## 💬 Interactive Mode Commands

When in interactive mode (`--interactive`):

- `help`, `h`, `?` - Show available commands
- `clear`, `cls` - Clear screen
- `config` - Show current configuration
- `exit`, `quit`, `bye`, `q` - Exit interactive mode

## 🧪 Testing

Run the comprehensive test suite:
```bash
go test -v
```

Tests cover:
- Configuration validation
- Path normalization
- Error handling
- Core functionality

## 📊 Examples

### Single Command Mode
```bash
# List files (default model)
./assistant "list files in current directory"

# With specific model
./assistant "show disk usage" "gpt-4.1"

# Weather tool
./assistant "what's the weather like in Tokyo"
```

### Interactive Mode
```bash
./assistant -i
🤖 Windows Automation Assistant (Interactive Mode)
Type 'exit', 'quit', or 'bye' to end the session
Type 'help' for available commands

You: list files in downloads folder
Assistant: [Lists files...]

You: create a backup script
Assistant: [Generates script...]

You: exit
Goodbye! 👋
```

### JSON Output Mode
```bash
./assistant --json "analyze this file"
# Outputs structured JSON response for programmatic use
# (Framework ready for structured data output)
```

## 📋 Requirements

- **GitHub Copilot CLI**: Installed and authenticated
- **Go**: 1.21+
- **Windows**: 11 environment