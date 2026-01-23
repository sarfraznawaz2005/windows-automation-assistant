package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	copilot "github.com/github/copilot-sdk/go"
)

// runInteractiveMode starts the interactive conversation loop
func runInteractiveMode(config *Config) {
	fmt.Println("Windows Automation Assistant (Interactive Mode)")
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

	// Cleanup function for graceful shutdown
	cleanup := func() {
		mu.Lock()
		defer mu.Unlock()
		stopThinking()
		if toolProgressStop != nil {
			toolProgressStop()
		}
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cleanup()
		fmt.Println("\nGoodbye!")
		os.Exit(0)
	}()

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
				fmt.Printf("Executing %s...\n", toolName)
				toolProgressStop = ShowToolExecution(toolName, config.Output.Spinner)
			}
		case "tool.execution_complete":
			// Stop the progress indicator
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
				fmt.Println("Tool execution completed")
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
					safeColor(colorRed), getUserFriendlyError(errors.New(*event.Data.Message), "Session"), safeColor(colorReset))
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
			fmt.Println("Goodbye!")
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
			fmt.Fprintf(os.Stderr, "%sError: %s%s\n", safeColor(colorRed), getUserFriendlyError(err, "Processing message"), safeColor(colorReset))
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
