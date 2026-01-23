package main

import (
	"os"
	"time"

	"github.com/briandowns/spinner"
)

// ProgressIndicator provides visual feedback for long-running operations
type ProgressIndicator struct {
	spinner *spinner.Spinner
	enabled bool
}

// NewProgressIndicator creates a new progress indicator
func NewProgressIndicator(message string, enabled bool) *ProgressIndicator {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond) // CharSets[14] = dots
	s.Suffix = " " + message
	s.Writer = os.Stderr
	return &ProgressIndicator{
		spinner: s,
		enabled: enabled,
	}
}

// Start begins showing the progress indicator
func (p *ProgressIndicator) Start() {
	if p.enabled {
		p.spinner.Start()
	}
}

// Stop ends the progress indicator
func (p *ProgressIndicator) Stop() {
	if p.spinner.Active() {
		p.spinner.Stop()
	}
}

// ShowProgress is a convenience function for quick progress indication
func ShowProgress(message string, enabled bool, operation func() error) error {
	indicator := NewProgressIndicator(message, enabled)
	indicator.Start()
	defer indicator.Stop()

	return operation()
}

// ShowToolExecution shows progress during tool execution
func ShowToolExecution(toolName string, enabled bool) func() {
	indicator := NewProgressIndicator("Executing "+toolName+"...", enabled)
	indicator.Start()

	return func() {
		indicator.Stop()
	}
}
