package main

import (
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

// IsMarkdown checks if text contains markdown syntax
func IsMarkdown(text string) bool {
	// Check for common markdown patterns
	markdownPatterns := []string{
		"**", "*", "_", "`", "# ", "## ", "### ", "#### ", "##### ", "###### ",
		"- ", "* ", "+ ", "1. ", "2. ", "3. ", "[",
		"```", "~~~", ">", "|", "---", "___", "***",
	}

	for _, pattern := range markdownPatterns {
		if contains(text, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// FormatAssistantResponse formats the assistant response with markdown if appropriate
func FormatAssistantResponse(response string, useMarkdown bool) string {
	if !useMarkdown {
		return response
	}

	// Check if response contains markdown
	if !IsMarkdown(response) {
		return response
	}

	// Convert to terminal-formatted text using glamour
	renderer := NewMarkdownRenderer()
	formatted, err := renderer.RenderToTerminal(response)
	if err != nil {
		// Fallback to original response if conversion fails
		return response
	}

	return formatted
}

// Global markdown renderer instance
var globalMarkdownRenderer *MarkdownRenderer

// init initializes the global markdown renderer
func init() {
	globalMarkdownRenderer = NewMarkdownRenderer()
}

// RenderMarkdown is a convenience function using the global renderer
func RenderMarkdown(markdown string) string {
	formatted, err := globalMarkdownRenderer.RenderToTerminal(markdown)
	if err != nil {
		return markdown // Return original if conversion fails
	}
	return formatted
}
