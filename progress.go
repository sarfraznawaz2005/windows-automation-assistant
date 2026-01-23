package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ProgressIndicator provides visual feedback for long-running operations
type ProgressIndicator struct {
	message  string
	active   bool
	stopChan chan bool
}

// NewProgressIndicator creates a new progress indicator
func NewProgressIndicator(message string) *ProgressIndicator {
	return &ProgressIndicator{
		message:  message,
		stopChan: make(chan bool, 1),
	}
}

// Start begins showing the progress indicator
func (p *ProgressIndicator) Start() {
	p.active = true
	go p.animate()
}

// Stop ends the progress indicator
func (p *ProgressIndicator) Stop() {
	if p.active {
		p.active = false
		p.stopChan <- true
		// Clear the line
		fmt.Print("\r" + strings.Repeat(" ", len(p.message)+10) + "\r")
	}
}

// animate runs the progress animation
func (p *ProgressIndicator) animate() {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	for p.active {
		select {
		case <-p.stopChan:
			return
		default:
			// Only show spinner if output is to terminal
			if isTerminal() {
				fmt.Printf("\r%s %s", spinner[i%len(spinner)], p.message)
			}
			i++
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// isTerminal checks if output is going to a terminal
func isTerminal() bool {
	// Simple check - in production, you'd use more sophisticated detection
	// For now, assume it's a terminal unless explicitly disabled
	return os.Getenv("NO_SPINNER") != "1"
}

// ShowProgress is a convenience function for quick progress indication
func ShowProgress(message string, operation func() error) error {
	indicator := NewProgressIndicator(message)
	indicator.Start()
	defer indicator.Stop()

	return operation()
}

// ShowToolExecution shows progress during tool execution
func ShowToolExecution(toolName string) func() {
	message := fmt.Sprintf("Executing %s...", toolName)
	indicator := NewProgressIndicator(message)
	indicator.Start()

	return func() {
		indicator.Stop()
		fmt.Printf("✓ %s completed\n", toolName)
	}
}
