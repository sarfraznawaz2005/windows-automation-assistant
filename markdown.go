package main

import (
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
)

// MarkdownRenderer handles markdown to terminal-formatted text conversion
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
}

// NewMarkdownRenderer creates a new terminal markdown renderer
func NewMarkdownRenderer() *MarkdownRenderer {
	// Create a terminal renderer with auto-detection and word wrapping
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),   // Auto-detect terminal theme
		glamour.WithWordWrap(100), // Reasonable width for wrapping
	)

	if err != nil {
		// Fallback to basic renderer if auto-detection fails
		r, _ = glamour.NewTermRenderer(
			glamour.WithWordWrap(100),
		)
	}

	return &MarkdownRenderer{renderer: r}
}

// RenderToTerminal converts markdown to terminal-formatted text
func (r *MarkdownRenderer) RenderToTerminal(markdown string) (string, error) {
	return r.renderer.Render(markdown)
}

// Global markdown renderer instance (lazily initialized)
var (
	globalMarkdownRenderer *MarkdownRenderer
	markdownOnce           sync.Once
)

// getMarkdownRenderer lazily initializes and returns the global markdown renderer
func getMarkdownRenderer() *MarkdownRenderer {
	markdownOnce.Do(func() {
		globalMarkdownRenderer = NewMarkdownRenderer()
	})
	return globalMarkdownRenderer
}

// RenderMarkdown is a convenience function using the global renderer
func RenderMarkdown(markdown string) string {
	renderer := getMarkdownRenderer()
	formatted, err := renderer.RenderToTerminal(markdown)
	if err != nil {
		return markdown // Return original if conversion fails
	}
	// Trim leading/trailing whitespace that glamour adds for styling
	return strings.TrimSpace(formatted)
}
