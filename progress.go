package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Unicode dot spinner characters (smooth animation)
var spinnerChars = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// ProgressIndicator provides visual feedback for long-running operations
type ProgressIndicator struct {
	message string
	enabled bool
	active  bool
	stop    chan struct{}
	done    chan struct{}
	mu      sync.Mutex
}

// NewProgressIndicator creates a new progress indicator
func NewProgressIndicator(message string, enabled bool) *ProgressIndicator {
	return &ProgressIndicator{
		message: message,
		enabled: enabled,
		active:  false,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// Start begins showing the progress indicator
func (p *ProgressIndicator) Start() {
	p.mu.Lock()
	if !p.enabled || p.active {
		p.mu.Unlock()
		return
	}
	p.active = true
	p.stop = make(chan struct{})
	p.done = make(chan struct{})
	p.mu.Unlock()

	go p.spin()
}

// spin runs the spinner animation in a goroutine
func (p *ProgressIndicator) spin() {
	defer close(p.done)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	i := 0
	// Hide cursor
	fmt.Fprint(os.Stderr, "\033[?25l")

	for {
		select {
		case <-p.stop:
			// Clear line and show cursor
			fmt.Fprint(os.Stderr, "\r\033[K\033[?25h")
			return
		case <-ticker.C:
			char := spinnerChars[i%len(spinnerChars)]
			fmt.Fprintf(os.Stderr, "\r%c %s", char, p.message)
			i++
		}
	}
}

// Stop ends the progress indicator
func (p *ProgressIndicator) Stop() {
	p.mu.Lock()
	if !p.active {
		p.mu.Unlock()
		return
	}
	p.active = false
	p.mu.Unlock()

	close(p.stop)
	<-p.done // Wait for goroutine to finish
}

// Active returns whether the spinner is currently running
func (p *ProgressIndicator) Active() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active
}

// ShowToolExecution shows progress during tool execution (only used in debug mode)
func ShowToolExecution(toolName string, enabled bool) func() {
	indicator := NewProgressIndicator("Executing "+toolName+"...", enabled)
	indicator.Start()

	return func() {
		indicator.Stop()
	}
}
