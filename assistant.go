package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	copilot "github.com/github/copilot-sdk/go"
)

// ANSI color codes for terminal output
const (
	red    = "\033[31m"
	reset  = "\033[0m"
	yellow = "\033[33m"
)

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

// CLI flags
var (
	interactive    = flag.Bool("interactive", false, "Enable interactive mode")
	i              = flag.Bool("i", false, "Enable interactive mode (short)")
	jsonOutput     = flag.Bool("json", false, "Output in JSON format")
	noMarkdown     = flag.Bool("no-markdown", false, "Disable markdown rendering")
	markdown       = flag.Bool("markdown", false, "Force enable markdown rendering")
	configPath     = flag.String("config", "", "Path to config file")
	generateConfig = flag.Bool("generate-config", false, "Generate default config file and exit")
)

// ErrorInfo holds error information with context
type ErrorInfo struct {
	Message  string
	File     string
	Line     int
	Function string
}

// handleError gracefully handles errors with user-friendly output
func handleError(err error, context string) {
	if err == nil {
		return
	}

	// Get caller information
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

	errorInfo := ErrorInfo{
		Message:  err.Error(),
		File:     file,
		Line:     line,
		Function: funcName,
	}

	// Show user-friendly error message
	fmt.Fprintf(os.Stderr, "%sError: %s%s\n", safeColor(red), getUserFriendlyError(err, context), safeColor(reset))

	// Show detailed error info for debugging (only in verbose mode or for developers)
	if os.Getenv("ASSISTANT_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "%s[DEBUG] %s:%d in %s: %s%s\n",
			safeColor(yellow), errorInfo.File, errorInfo.Line, errorInfo.Function, err.Error(), safeColor(reset))
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

// generateDefaultConfig creates a default configuration file
func generateDefaultConfig() error {
	config := DefaultConfig()
	return SaveConfig(config, "config.yaml")
}

// runInteractiveMode starts the interactive conversation loop
func runInteractiveMode(config *Config) {
	fmt.Println("🤖 Windows Automation Assistant (Interactive Mode)")
	fmt.Println("Type 'exit', 'quit', or 'bye' to end the session")
	fmt.Println("Type 'help' for available commands")
	fmt.Println()

	// Load custom tools
	customTools, err := loadCustomTools(config)
	if err != nil {
		handleError(err, "Loading custom tools")
	}

	// Create client
	client := copilot.NewClient(&copilot.ClientOptions{
		LogLevel:    config.ClientOptions.LogLevel,
		CLIPath:     config.ClientOptions.CLIPath,
		AutoStart:   config.ClientOptions.AutoStart,
		AutoRestart: config.ClientOptions.AutoRestart,
	})

	if err := client.Start(); err != nil {
		handleError(err, "Starting Copilot client")
	}
	defer client.Stop()

	// Define the assistant agent using config
	assistantAgent := copilot.CustomAgentConfig{
		Name:        "assistant",
		DisplayName: "Personal Assistant",
		Description: "A personal assistant for Windows automation tasks",
		Prompt:      config.SystemPrompt,
		Infer:       copilot.Bool(true),
	}

	// Create a session with the assistant agent and custom tools
	sessionConfig := &copilot.SessionConfig{
		Model:        config.Model,
		CustomAgents: []copilot.CustomAgentConfig{assistantAgent},
		Streaming:    true,
	}

	// Add custom tools if any were loaded
	if len(customTools) > 0 {
		sessionConfig.Tools = customTools
	}

	session, err := client.CreateSession(sessionConfig)
	if err != nil {
		handleError(err, "Creating session")
	}
	defer session.Destroy()

	// Interactive conversation loop
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("You: ")

	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())

		// Handle exit commands
		if isExitCommand(input) {
			fmt.Println("Goodbye! 👋")
			break
		}

		// Handle special commands
		if handled := handleSpecialCommand(input); handled {
			fmt.Print("You: ")
			continue
		}

		// Process with Copilot
		if err := processInteractiveMessage(session, input, config); err != nil {
			fmt.Fprintf(os.Stderr, "%sError: %s%s\n", red, getUserFriendlyError(err, "Processing message"), reset)
		}

		fmt.Print("You: ")
	}

	if err := scanner.Err(); err != nil {
		handleError(err, "Reading input")
	}
}

// isExitCommand checks if the input is an exit command
func isExitCommand(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "exit" || input == "quit" || input == "bye" || input == "q"
}

// handleSpecialCommand handles special commands like help, clear, etc.
func handleSpecialCommand(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "help", "h", "?":
		showHelp()
		return true
	case "clear", "cls":
		// Clear screen (basic implementation)
		fmt.Print("\033[2J\033[1;1H") // ANSI clear screen
		return true
	case "config":
		showCurrentConfig()
		return true
	default:
		return false
	}
}

// showHelp displays available commands
func showHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  help, h, ?     Show this help message")
	fmt.Println("  clear, cls     Clear the screen")
	fmt.Println("  config         Show current configuration")
	fmt.Println("  exit, quit, bye, q    Exit interactive mode")
	fmt.Println("\nJust type your automation request and press Enter!")
	fmt.Println()
}

// showCurrentConfig displays current configuration (simplified)
func showCurrentConfig() {
	fmt.Println("\nCurrent configuration:")
	fmt.Println("  Model: gpt-4.1")
	fmt.Println("  Debug: disabled")
	fmt.Println("  Tools: enabled")
	fmt.Println()
}

// processInteractiveMessage sends a message and handles the response
func processInteractiveMessage(session *copilot.Session, message string, config *Config) error {
	done := make(chan bool)
	var toolProgressStop func()

	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		case "assistant.message_delta":
			if event.Data.DeltaContent != nil {
				content := *event.Data.DeltaContent
				if config.Output.Markdown {
					content = RenderMarkdown(content)
				}
				fmt.Print(content)
			}
		case "assistant.message":
			if event.Data.Content != nil {
				content := *event.Data.Content
				if config.Output.Markdown {
					content = RenderMarkdown(content)
				}
				fmt.Println(content)
			}
		case "tool.execution_start":
			// Stop any existing progress indicator
			if toolProgressStop != nil {
				toolProgressStop()
			}
			// Start new progress indicator for tool execution
			if event.Data.ToolRequests != nil && len(event.Data.ToolRequests) > 0 {
				toolName := event.Data.ToolRequests[0].Name
				fmt.Printf("🔧 Executing %s...\n", toolName)
				toolProgressStop = ShowToolExecution(toolName)
			}
		case "tool.execution_complete":
			// Stop the progress indicator
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
				fmt.Println("✅ Tool execution completed")
			}
		case "session.idle":
			// Ensure any remaining progress indicator is stopped
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
			}
			close(done)
		case "session.error":
			// Stop any progress indicator on error
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
			}
			if event.Data.Message != nil {
				fmt.Fprintf(os.Stderr, "%sSession Error: %s%s\n",
					safeColor(red), getUserFriendlyError(errors.New(*event.Data.Message), "Session"), safeColor(reset))
			}
			close(done)
		}
	})

	// Send the message
	_, err := session.Send(copilot.MessageOptions{
		Prompt: message,
	})
	if err != nil {
		return err
	}

	// Wait for completion
	<-done
	return nil
}

func main() {
	flag.Usage = func() {
		// Get executable name without full path for cleaner output
		execName := filepath.Base(os.Args[0])
		if runtime.GOOS == "windows" && strings.HasSuffix(execName, ".exe") {
			execName = strings.TrimSuffix(execName, ".exe")
		}

		fmt.Fprintf(os.Stderr, "Usage: %s [options] [prompt] [model]\n", execName)
		fmt.Fprintf(os.Stderr, "\nWindows Automation Assistant - AI-powered Windows task automation\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s \"list files in current directory\"\n", execName)
		fmt.Fprintf(os.Stderr, "  %s -interactive\n", execName)
		fmt.Fprintf(os.Stderr, "  %s --json \"analyze this file\" \"gpt-4.1\"\n", execName)
		fmt.Fprintf(os.Stderr, "  %s --markdown \"create a table of processes\"\n", execName)
		fmt.Fprintf(os.Stderr, "  %s --no-markdown \"simple text only\"\n", execName)
		fmt.Fprintf(os.Stderr, "  %s --generate-config\n", execName)
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  ASSISTANT_DEBUG=1     Show detailed error information with file/line numbers\n")
		fmt.Fprintf(os.Stderr, "  NO_SPINNER=1          Disable progress spinner animations\n")
		fmt.Fprintf(os.Stderr, "\nFor more information, see README.md\n")
	}

	flag.Parse()

	// Handle generate config command
	if *generateConfig {
		if err := generateDefaultConfig(); err != nil {
			handleError(err, "Generating config")
		}
		fmt.Println("Default config file generated successfully")
		return
	}

	// Load configuration
	config, err := LoadConfig(*configPath)
	if err != nil {
		handleError(err, "Loading configuration")
	}

	// Override config with CLI flags
	if *noMarkdown {
		config.Output.Markdown = false
	} else if *markdown {
		config.Output.Markdown = true
	}

	// Override config with environment variables
	if debugEnv := os.Getenv("ASSISTANT_DEBUG"); debugEnv == "1" {
		config.Debug = true
	}

	// Validate configuration
	if err := ValidateConfig(config); err != nil {
		handleError(err, "Configuration validation")
	}

	// Determine mode
	isInteractive := *interactive || *i

	// Get prompt and model from arguments or interactive input
	var prompt, model string

	if isInteractive {
		// Interactive mode - no arguments needed
		model = config.Model
		runInteractiveMode(config)
		return
	} else {
		// Single command mode
		args := flag.Args()
		if len(args) < 1 {
			flag.Usage()
			os.Exit(1)
		}

		prompt = args[0]
		model = config.Model // default from config
		if len(args) >= 2 {
			model = args[1] // override from command line
		}
	}

	// Create client with config
	client := copilot.NewClient(&copilot.ClientOptions{
		LogLevel:    config.ClientOptions.LogLevel,
		CLIPath:     config.ClientOptions.CLIPath,
		AutoStart:   config.ClientOptions.AutoStart,
		AutoRestart: config.ClientOptions.AutoRestart,
	})

	// Start the client
	if err := client.Start(); err != nil {
		handleError(err, "Starting Copilot client")
	}
	defer client.Stop()

	// Load custom tools
	customTools, err := loadCustomTools(config)
	if err != nil {
		handleError(err, "Loading custom tools")
	}

	// Define the assistant agent using config
	assistantAgent := copilot.CustomAgentConfig{
		Name:        "assistant",
		DisplayName: "Personal Assistant",
		Description: "A personal assistant for Windows automation tasks",
		Prompt:      config.SystemPrompt,
		Infer:       copilot.Bool(true),
	}

	// Create a session with the assistant agent and custom tools
	sessionConfig := &copilot.SessionConfig{
		Model:        model,
		CustomAgents: []copilot.CustomAgentConfig{assistantAgent},
		Streaming:    true,
	}

	// Add custom tools if any were loaded
	if len(customTools) > 0 {
		sessionConfig.Tools = customTools
	}

	session, err := client.CreateSession(sessionConfig)
	if err != nil {
		handleError(err, "Creating session")
	}
	defer session.Destroy()

	// Set up event handler for streaming responses
	done := make(chan bool)
	var toolProgressStop func()
	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		case "assistant.message_delta":
			if event.Data.DeltaContent != nil {
				content := *event.Data.DeltaContent
				if config.Output.Markdown {
					content = RenderMarkdown(content)
				}
				fmt.Print(content)
			}
		case "assistant.message":
			if event.Data.Content != nil {
				content := *event.Data.Content
				if config.Output.Markdown {
					content = RenderMarkdown(content)
				}
				fmt.Println(content)
			}
		case "tool.execution_start":
			// Stop any existing progress indicator
			if toolProgressStop != nil {
				toolProgressStop()
			}
			// Start new progress indicator for tool execution
			if event.Data.ToolRequests != nil && len(event.Data.ToolRequests) > 0 {
				toolName := event.Data.ToolRequests[0].Name
				fmt.Printf("🔧 Executing %s...\n", toolName)
				toolProgressStop = ShowToolExecution(toolName)
			}
		case "tool.execution_complete":
			// Stop the progress indicator
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
				fmt.Println("✅ Tool execution completed")
			}
		case "session.idle":
			// Ensure any remaining progress indicator is stopped
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
			}
			close(done)
		case "session.error":
			// Stop any progress indicator on error
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
			}
			if event.Data.Message != nil {
				// Show session errors in red with user-friendly formatting
				fmt.Fprintf(os.Stderr, "%sSession Error: %s%s\n",
					safeColor(red), getUserFriendlyError(errors.New(*event.Data.Message), "Session"), safeColor(reset))

				// Show debug info if enabled
				if os.Getenv("ASSISTANT_DEBUG") == "1" {
					fmt.Fprintf(os.Stderr, "%s[DEBUG] Raw session error: %s%s\n",
						safeColor(yellow), *event.Data.Message, safeColor(reset))
				}
			}
			close(done)
		}
	})

	// Send the prompt
	_, err = session.Send(copilot.MessageOptions{
		Prompt: prompt,
	})
	if err != nil {
		handleError(err, "Sending message")
	}

	// Wait for completion
	<-done
}
