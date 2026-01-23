package main

import (
	"regexp"
	"strings"
	"sync"
)

// ANSI codes for markdown styling
const (
	mdBold      = "\033[1m"
	mdItalic    = "\033[3m"
	mdCode      = "\033[48;5;236m\033[38;5;252m" // Dark gray bg, light text
	mdCodeBlock = "\033[48;5;235m"               // Slightly darker bg for code blocks
	mdHeader    = "\033[1;36m"                   // Bold cyan for headers
	mdLink      = "\033[4;34m"                   // Underline blue for links
	mdList      = "\033[33m"                     // Yellow for list markers
	mdReset     = "\033[0m"
)

// MarkdownRenderer handles markdown to terminal-formatted text conversion
type MarkdownRenderer struct {
	// Compiled regex patterns for performance
	boldPattern       *regexp.Regexp
	italicPattern     *regexp.Regexp
	inlineCodePattern *regexp.Regexp
	linkPattern       *regexp.Regexp
	headerPattern     *regexp.Regexp
}

// NewMarkdownRenderer creates a new terminal markdown renderer
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{
		boldPattern:       regexp.MustCompile(`\*\*(.+?)\*\*|__(.+?)__`),
		italicPattern:     regexp.MustCompile(`(?:^|[^*])\*([^*]+?)\*(?:[^*]|$)|(?:^|[^_])_([^_]+?)_(?:[^_]|$)`),
		inlineCodePattern: regexp.MustCompile("`([^`]+)`"),
		linkPattern:       regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`),
		headerPattern:     regexp.MustCompile(`^(#{1,6})\s+(.+)$`),
	}
}

// RenderToTerminal converts markdown to terminal-formatted text
func (r *MarkdownRenderer) RenderToTerminal(markdown string) (string, error) {
	lines := strings.Split(markdown, "\n")
	var result []string
	inCodeBlock := false
	codeBlockLang := ""
	var codeBlockContent []string

	for _, line := range lines {
		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// End code block - render collected content
				result = append(result, r.renderCodeBlock(codeBlockContent, codeBlockLang))
				inCodeBlock = false
				codeBlockLang = ""
				codeBlockContent = nil
			} else {
				// Start code block
				inCodeBlock = true
				codeBlockLang = strings.TrimPrefix(line, "```")
				codeBlockContent = []string{}
			}
			continue
		}

		if inCodeBlock {
			codeBlockContent = append(codeBlockContent, line)
			continue
		}

		// Process normal lines
		result = append(result, r.renderLine(line))
	}

	// Handle unclosed code block
	if inCodeBlock && len(codeBlockContent) > 0 {
		result = append(result, r.renderCodeBlock(codeBlockContent, codeBlockLang))
	}

	return strings.Join(result, "\n"), nil
}

// renderCodeBlock renders a code block with colored background and indentation
func (r *MarkdownRenderer) renderCodeBlock(lines []string, lang string) string {
	if len(lines) == 0 {
		return ""
	}

	var sb strings.Builder

	// Find max line length for consistent background
	maxLen := 0
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}
	// Minimum width and add padding
	if maxLen < 40 {
		maxLen = 40
	}
	maxLen += 4 // padding

	// Top border
	sb.WriteString("  ")
	sb.WriteString(mdCodeBlock)
	sb.WriteString(strings.Repeat(" ", maxLen))
	sb.WriteString(mdReset)
	sb.WriteString("\n")

	// Code lines with background
	for _, line := range lines {
		sb.WriteString("  ")
		sb.WriteString(mdCodeBlock)
		sb.WriteString("  ") // left padding
		sb.WriteString(line)
		// Pad to max length for consistent background
		padding := maxLen - len(line) - 2
		if padding > 0 {
			sb.WriteString(strings.Repeat(" ", padding))
		}
		sb.WriteString(mdReset)
		sb.WriteString("\n")
	}

	// Bottom border
	sb.WriteString("  ")
	sb.WriteString(mdCodeBlock)
	sb.WriteString(strings.Repeat(" ", maxLen))
	sb.WriteString(mdReset)

	return sb.String()
}

// renderLine renders a single line of markdown
func (r *MarkdownRenderer) renderLine(line string) string {
	// Check for headers first
	if match := r.headerPattern.FindStringSubmatch(line); match != nil {
		level := len(match[1])
		text := match[2]
		// Apply inline formatting to header text
		text = r.applyInlineFormatting(text)
		prefix := strings.Repeat("#", level) + " "
		return mdHeader + prefix + text + mdReset
	}

	// Check for unordered list items
	trimmed := strings.TrimLeft(line, " \t")
	indent := len(line) - len(trimmed)
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		content := trimmed[2:]
		content = r.applyInlineFormatting(content)
		return strings.Repeat(" ", indent) + mdList + "• " + mdReset + content
	}

	// Check for ordered list items
	if match := regexp.MustCompile(`^(\d+)\.\s+(.+)$`).FindStringSubmatch(trimmed); match != nil {
		num := match[1]
		content := r.applyInlineFormatting(match[2])
		return strings.Repeat(" ", indent) + mdList + num + ". " + mdReset + content
	}

	// Regular line - apply inline formatting
	return r.applyInlineFormatting(line)
}

// applyInlineFormatting applies bold, italic, code, and link formatting
func (r *MarkdownRenderer) applyInlineFormatting(text string) string {
	// Order matters: process code first (to avoid formatting inside code)
	// Then links, bold, italic

	// Inline code
	text = r.inlineCodePattern.ReplaceAllString(text, mdCode+" $1 "+mdReset)

	// Links [text](url) -> text (url)
	text = r.linkPattern.ReplaceAllString(text, mdLink+"$1"+mdReset+" ($2)")

	// Bold **text** or __text__
	text = r.boldPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Extract content between markers
		content := strings.TrimPrefix(match, "**")
		content = strings.TrimSuffix(content, "**")
		content = strings.TrimPrefix(content, "__")
		content = strings.TrimSuffix(content, "__")
		return mdBold + content + mdReset
	})

	// Italic *text* or _text_ (be careful not to match bold markers)
	// Simple approach: match single asterisks/underscores not adjacent to others
	text = r.renderItalic(text)

	return text
}

// renderItalic handles italic formatting carefully to avoid conflicts with bold
func (r *MarkdownRenderer) renderItalic(text string) string {
	// Match *text* but not **text** - use simple state machine approach
	var result strings.Builder
	runes := []rune(text)
	i := 0

	for i < len(runes) {
		// Check for single * or _ that's not part of ** or __
		if (runes[i] == '*' || runes[i] == '_') && !r.isDoubleMarker(runes, i) {
			marker := runes[i]
			// Find closing marker
			start := i + 1
			end := r.findClosingMarker(runes, start, marker)
			if end > start {
				result.WriteString(mdItalic)
				result.WriteString(string(runes[start:end]))
				result.WriteString(mdReset)
				i = end + 1
				continue
			}
		}
		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

// isDoubleMarker checks if position i is part of ** or __
func (r *MarkdownRenderer) isDoubleMarker(runes []rune, i int) bool {
	if i+1 < len(runes) && runes[i+1] == runes[i] {
		return true
	}
	if i > 0 && runes[i-1] == runes[i] {
		return true
	}
	return false
}

// findClosingMarker finds the closing italic marker
func (r *MarkdownRenderer) findClosingMarker(runes []rune, start int, marker rune) int {
	for i := start; i < len(runes); i++ {
		if runes[i] == marker && !r.isDoubleMarker(runes, i) {
			return i
		}
	}
	return -1
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
	// Trim leading/trailing whitespace
	return strings.TrimSpace(formatted)
}
