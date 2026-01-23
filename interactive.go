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

	// Create client
	client := copilot.NewClient(&copilot.ClientOptions{
		LogLevel:    config.ClientOptions.LogLevel,
		CLIPath:     config.ClientOptions.CLIPath,
		AutoStart:   config.ClientOptions.AutoStart,
		AutoRestart: config.ClientOptions.AutoRestart,
	})

	// Start client and load tools in parallel for faster startup
	var wg sync.WaitGroup
	var clientErr error
	var customTools []copilot.Tool
	var toolsErr error

	wg.Add(2)

	go func() {
		defer wg.Done()
		clientErr = client.Start()
	}()

	go func() {
		defer wg.Done()
		customTools, toolsErr = loadCustomTools(config)
	}()

	wg.Wait()

	// Handle errors after parallel operations complete
	if clientErr != nil {
		handleError(clientErr, "Starting Copilot client")
	}
	defer client.Stop()

	if toolsErr != nil {
		handleError(toolsErr, "Loading custom tools")
	}

	// Create a session with system message and custom tools
	sessionConfig := &copilot.SessionConfig{
		Model:     config.Model,
		Streaming: config.Output.Streaming,
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
		streamedContent   bool       // track if we've already streamed content
		mu                sync.Mutex // protect shared state
	)
	fullContent.Grow(4096) // Pre-allocate 4KB for typical responses

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
				if !config.Output.Streaming {
					// Non-streaming mode: collect content for final output
					fullContent.WriteString(content)
				} else {
					// Streaming mode - stop spinner and print immediately
					// Only print if content is not empty (skip empty deltas)
					if content != "" {
						stopThinking()
						fmt.Print(content)
						streamedContent = true
					}
				}
			}
		case "assistant.message":
			// Stop thinking indicator before showing final output
			stopThinking()
			if !config.Output.Streaming && fullContent.Len() > 0 {
				// Non-streaming mode: render collected content
				content := fullContent.String()
				if config.Output.Markdown {
					content = RenderMarkdown(content)
				}
				fmt.Println(content)
			} else if streamedContent {
				// Streaming mode completed - just add newline
				fmt.Println()
			} else if event.Data.Content != nil {
				// Non-streaming response (no deltas received)
				content := *event.Data.Content
				if config.Output.Markdown {
					content = RenderMarkdown(content)
				}
				fmt.Println(content)
			}
		case "tool.execution_start":
			// Start new progress indicator for tool execution - check both ToolName and ToolRequests
			var toolName string
			if event.Data.ToolName != nil {
				toolName = *event.Data.ToolName
			} else if event.Data.ToolRequests != nil && len(event.Data.ToolRequests) > 0 {
				toolName = event.Data.ToolRequests[0].Name
			}
			// Skip internal tools like report_intent
			if toolName != "" && toolName != "report_intent" {
				if config.Debug {
					// Only show tool execution details in debug mode
					stopThinking()
					if toolProgressStop != nil {
						toolProgressStop()
					}
					fmt.Printf("Executing Tool: %s...\n", toolName)
					toolProgressStop = ShowToolExecution(toolName, config.Output.Spinner)
				}
			}
		case "tool.execution_complete":
			// Stop the tool progress indicator (only active in debug mode)
			if toolProgressStop != nil {
				toolProgressStop()
				toolProgressStop = nil
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
		streamedContent = false
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
	fmt.Printf("  Streaming: %v\n", config.Output.Streaming)
	fmt.Printf("  Tools enabled: %v\n", config.Tools.Enabled)
	if config.Tools.Enabled && len(config.Tools.EnabledTools) > 0 {
		fmt.Printf("  Enabled tools: %v\n", config.Tools.EnabledTools)
	}
	fmt.Println()
}
