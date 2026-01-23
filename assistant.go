package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

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
	noSpinner      = flag.Bool("no-spinner", false, "Disable loading spinner")
	showSpinner    = flag.Bool("spinner", false, "Force enable loading spinner")
	configPath     = flag.String("config", "", "Path to config file")
	generateConfig = flag.Bool("generate-config", false, "Generate default config file and exit")
)

// JSONResponse is the structure for JSON output mode
type JSONResponse struct {
	Success  bool     `json:"success"`
	Response string   `json:"response,omitempty"`
	Error    string   `json:"error,omitempty"`
	Model    string   `json:"model,omitempty"`
	Tools    []string `json:"tools_used,omitempty"`
}

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

	// Create a session with system message and custom tools
	sessionConfig := &copilot.SessionConfig{
		Model:     config.Model,
		Streaming: true,
		SystemMessage: &copilot.SystemMessageConfig{
			Content: config.SystemPrompt,
		},
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

	// Shared state for event handling
	var (
		done              chan struct{}
		toolProgressStop  func()
		fullContent       strings.Builder
		thinkingIndicator *ProgressIndicator
		thinkingStopped   bool
		mu                sync.Mutex // protect shared state
	)

	// Helper to stop thinking indicator
	stopThinking := func() {
		if !thinkingStopped && thinkingIndicator != nil {
			thinkingIndicator.Stop()
			thinkingStopped = true
		}
	}

	// Set up event handler ONCE for the entire session
	session.On(func(event copilot.SessionEvent) {
		mu.Lock()
		defer mu.Unlock()

		// Ignore events if we're not waiting for a response
		if done == nil {
			return
		}

		switch event.Type {
		case "assistant.message_delta":
			if event.Data.DeltaContent != nil {
				content := *event.Data.DeltaContent
				if config.Output.Markdown {
					// Collect content for final markdown rendering
					fullContent.WriteString(content)
				} else {
					// No markdown - stop spinner and print immediately
					stopThinking()
					fmt.Print(content)
				}
			}
		case "assistant.message":
			// Stop thinking indicator before showing final output
			stopThinking()
			if config.Output.Markdown && fullContent.Len() > 0 {
				// Render collected content as markdown
				fmt.Println(RenderMarkdown(fullContent.String()))
			} else if event.Data.Content != nil && fullContent.Len() == 0 {
				// Non-streaming response
				content := *event.Data.Content
				if config.Output.Markdown {
					content = RenderMarkdown(content)
				}
				fmt.Println(content)
			} else {
				// Streaming without markdown - just add newline
				fmt.Println()
			}
		case "tool.execution_start":
			// Stop thinking indicator before tool execution
			stopThinking()
			// Stop any existing progress indicator
			if toolProgressStop != nil {
				toolProgressStop()
			}
			// Start new progress indicator for tool execution
			if event.Data.ToolRequests != nil && len(event.Data.ToolRequests) > 0 {
				toolName := event.Data.ToolRequests[0].Name
				fmt.Printf("🔧 Executing %s...\n", toolName)
				toolProgressStop = ShowToolExecution(toolName, config.Output.Spinner)
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
			stopThinking()
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
			}
			if done != nil {
				close(done)
				done = nil
			}
		case "session.error":
			// Stop any progress indicator on error
			stopThinking()
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
			}
			if event.Data.Message != nil {
				fmt.Fprintf(os.Stderr, "%sSession Error: %s%s\n",
					safeColor(red), getUserFriendlyError(errors.New(*event.Data.Message), "Session"), safeColor(reset))
			}
			if done != nil {
				close(done)
				done = nil
			}
		}
	})

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
		if handled := handleSpecialCommand(input, config); handled {
			fmt.Print("You: ")
			continue
		}

		// Skip empty input
		if input == "" {
			fmt.Print("You: ")
			continue
		}

		// Reset state for new message
		mu.Lock()
		done = make(chan struct{})
		fullContent.Reset()
		thinkingStopped = false
		thinkingIndicator = NewProgressIndicator("Thinking...", config.Output.Spinner)
		thinkingIndicator.Start()
		currentDone := done
		mu.Unlock()

		// Send the message
		_, err := session.Send(copilot.MessageOptions{
			Prompt: input,
		})
		if err != nil {
			mu.Lock()
			thinkingIndicator.Stop()
			if done != nil {
				close(done)
				done = nil
			}
			mu.Unlock()
			fmt.Fprintf(os.Stderr, "%sError: %s%s\n", safeColor(red), getUserFriendlyError(err, "Processing message"), safeColor(reset))
			fmt.Print("You: ")
			continue
		}

		// Wait for completion
		<-currentDone

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
func handleSpecialCommand(input string, config *Config) bool {
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
		showCurrentConfig(config)
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

// showCurrentConfig displays current configuration
func showCurrentConfig(config *Config) {
	fmt.Println("\nCurrent configuration:")
	fmt.Printf("  Model: %s\n", config.Model)
	fmt.Printf("  Debug: %v\n", config.Debug)
	fmt.Printf("  Markdown: %v\n", config.Output.Markdown)
	fmt.Printf("  Spinner: %v\n", config.Output.Spinner)
	fmt.Printf("  Tools enabled: %v\n", config.Tools.Enabled)
	if config.Tools.Enabled {
		fmt.Printf("  Tools directory: %s\n", config.Tools.Directory)
	}
	fmt.Println()
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

	if *noSpinner {
		config.Output.Spinner = false
	} else if *showSpinner {
		config.Output.Spinner = true
	}

	// JSON output mode
	if *jsonOutput {
		config.Output.JSON = true
		config.Output.Markdown = false // Disable markdown for clean JSON
		config.Output.Spinner = false  // Disable spinner for clean JSON
	}

	// Override config with environment variables
	if debugEnv := os.Getenv("ASSISTANT_DEBUG"); debugEnv == "1" {
		config.Debug = true
	}
	if os.Getenv("NO_SPINNER") == "1" {
		config.Output.Spinner = false
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
		// Debug: show which model is being used
		if config.Debug {
			fmt.Fprintf(os.Stderr, "%s[DEBUG] Using model: %s%s\n", safeColor(yellow), model, safeColor(reset))
		}
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

	// Debug: show which model is being used
	if config.Debug {
		fmt.Fprintf(os.Stderr, "%s[DEBUG] Using model: %s%s\n", safeColor(yellow), model, safeColor(reset))
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

	// Create a session with system message and custom tools
	sessionConfig := &copilot.SessionConfig{
		Model:     model,
		Streaming: true,
		SystemMessage: &copilot.SystemMessageConfig{
			Content: config.SystemPrompt,
		},
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
	var fullContent strings.Builder // collect streamed content
	var toolsUsed []string          // track tools used for JSON output
	var sessionError string         // track session errors for JSON output
	var thinkingIndicator *ProgressIndicator
	thinkingStopped := false

	// Helper to stop thinking indicator
	stopThinking := func() {
		if !thinkingStopped && thinkingIndicator != nil {
			thinkingIndicator.Stop()
			thinkingStopped = true
		}
	}

	// Helper to output JSON response
	outputJSON := func(success bool, response, errMsg string) {
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

	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		case "assistant.message_delta":
			if event.Data.DeltaContent != nil {
				content := *event.Data.DeltaContent
				if config.Output.JSON || config.Output.Markdown {
					// Collect content for final output
					fullContent.WriteString(content)
				} else {
					// No markdown/JSON - stop spinner and print immediately
					stopThinking()
					fmt.Print(content)
				}
			}
		case "assistant.message":
			// Stop thinking indicator before showing final output
			stopThinking()
			if config.Output.JSON {
				// JSON output handled at session.idle
			} else if config.Output.Markdown && fullContent.Len() > 0 {
				// Render collected content as markdown
				fmt.Println(RenderMarkdown(fullContent.String()))
			} else if event.Data.Content != nil && fullContent.Len() == 0 {
				// Non-streaming response
				content := *event.Data.Content
				if config.Output.Markdown {
					content = RenderMarkdown(content)
				}
				fmt.Println(content)
			} else {
				// Streaming without markdown - just add newline
				fmt.Println()
			}
		case "tool.execution_start":
			// Stop thinking indicator before tool execution
			stopThinking()
			// Track tools used
			if event.Data.ToolRequests != nil && len(event.Data.ToolRequests) > 0 {
				toolName := event.Data.ToolRequests[0].Name
				toolsUsed = append(toolsUsed, toolName)
				if !config.Output.JSON {
					// Stop any existing progress indicator
					if toolProgressStop != nil {
						toolProgressStop()
					}
					fmt.Printf("🔧 Executing %s...\n", toolName)
					toolProgressStop = ShowToolExecution(toolName, config.Output.Spinner)
				}
			}
		case "tool.execution_complete":
			// Stop the progress indicator
			if !config.Output.JSON && toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
				fmt.Println("✅ Tool execution completed")
			}
		case "session.idle":
			// Ensure any remaining progress indicator is stopped
			stopThinking()
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
			}
			// Output JSON if enabled
			if config.Output.JSON {
				if sessionError != "" {
					outputJSON(false, "", sessionError)
				} else {
					outputJSON(true, fullContent.String(), "")
				}
			}
			close(done)
		case "session.error":
			// Stop any progress indicator on error
			stopThinking()
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
			}
			if event.Data.Message != nil {
				sessionError = *event.Data.Message
				if !config.Output.JSON {
					// Show session errors in red with user-friendly formatting
					fmt.Fprintf(os.Stderr, "%sSession Error: %s%s\n",
						safeColor(red), getUserFriendlyError(errors.New(*event.Data.Message), "Session"), safeColor(reset))

					// Show debug info if enabled
					if os.Getenv("ASSISTANT_DEBUG") == "1" {
						fmt.Fprintf(os.Stderr, "%s[DEBUG] Raw session error: %s%s\n",
							safeColor(yellow), *event.Data.Message, safeColor(reset))
					}
				}
			}
			close(done)
		}
	})

	// Start "Thinking..." indicator
	thinkingIndicator = NewProgressIndicator("Thinking...", config.Output.Spinner)
	thinkingIndicator.Start()

	// Send the prompt
	_, err = session.Send(copilot.MessageOptions{
		Prompt: prompt,
	})
	if err != nil {
		thinkingIndicator.Stop()
		handleError(err, "Sending message")
	}

	// Wait for completion
	<-done
}
