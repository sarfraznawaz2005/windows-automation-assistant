package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	copilot "github.com/github/copilot-sdk/go"
)

// runSingleCommand executes a single prompt and exits
func runSingleCommand(config *Config, prompt, model string) {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

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

	// Cleanup function for graceful shutdown
	cleanup := func() {
		stopThinking()
		if toolProgressStop != nil {
			toolProgressStop()
		}
	}

	// Handle interrupt signal in a goroutine
	go func() {
		<-sigChan
		cleanup()
		fmt.Println("\nInterrupted. Exiting...")
		os.Exit(0)
	}()

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
					fmt.Printf("Executing %s...\n", toolName)
					toolProgressStop = ShowToolExecution(toolName, config.Output.Spinner)
				}
			}
		case "tool.execution_complete":
			// Stop the progress indicator
			if !config.Output.JSON && toolProgressStop != nil {
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
			// Output JSON if enabled
			if config.Output.JSON {
				if sessionError != "" {
					outputJSON(false, "", sessionError, model, toolsUsed)
				} else {
					outputJSON(true, fullContent.String(), "", model, toolsUsed)
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
						safeColor(colorRed), getUserFriendlyError(errors.New(*event.Data.Message), "Session"), safeColor(colorReset))

					// Show debug info if enabled
					if os.Getenv("ASSISTANT_DEBUG") == "1" {
						fmt.Fprintf(os.Stderr, "%s[DEBUG] Raw session error: %s%s\n",
							safeColor(colorYellow), *event.Data.Message, safeColor(colorReset))
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
