// Grep tool - searches file contents using regex patterns
package usertools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	RegisterLazy(ToolDefinition{
		Name:        "grep",
		Description: "Search for text patterns in files using regex. Can search a single file or recursively search a directory. Returns matching lines with file paths and line numbers.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Regular expression pattern to search for",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File or directory path to search in (default: current directory)",
				},
				"include": map[string]interface{}{
					"type":        "string",
					"description": "Glob pattern to filter files (e.g. \"*.go\", \"*.txt\"). Only used when searching directories.",
				},
				"ignore_case": map[string]interface{}{
					"type":        "boolean",
					"description": "Case-insensitive search (default: false)",
				},
				"context_lines": map[string]interface{}{
					"type":        "number",
					"description": "Number of context lines to show before and after each match (default: 0)",
				},
			},
			"required": []string{"pattern"},
		},
		Loader: func() ToolHandler {
			return grepHandler
		},
	})
}

type grepParams struct {
	Pattern      string  `json:"pattern"`
	Path         string  `json:"path"`
	Include      string  `json:"include"`
	IgnoreCase   bool    `json:"ignore_case"`
	ContextLines float64 `json:"context_lines"`
}

type grepMatch struct {
	File    string
	LineNum int
	Line    string
}

func grepHandler(invocation ToolInvocation) (ToolResult, error) {
	var params grepParams
	if err := MapToStruct(invocation.Arguments, &params); err != nil {
		return ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Pattern == "" {
		return ToolResult{
			TextResultForLLM: "Error: pattern parameter is required",
			ResultType:       "error",
		}, nil
	}

	if params.Path == "" {
		params.Path = "."
	}

	// Compile regex
	regexPattern := params.Pattern
	if params.IgnoreCase {
		regexPattern = "(?i)" + regexPattern
	}
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error: invalid regex pattern: %v", err),
			ResultType:       "error",
		}, nil
	}

	contextLines := int(params.ContextLines)

	info, err := os.Stat(params.Path)
	if err != nil {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error: %v", err),
			ResultType:       "error",
		}, nil
	}

	var matches []grepMatch
	const maxMatches = 500
	filesSearched := 0

	if info.IsDir() {
		// Search directory recursively
		filepath.Walk(params.Path, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				if fi != nil && fi.IsDir() && strings.HasPrefix(fi.Name(), ".") && path != params.Path {
					return filepath.SkipDir
				}
				return nil
			}
			if len(matches) >= maxMatches {
				return filepath.SkipAll
			}

			// Filter by include pattern
			if params.Include != "" {
				matched, _ := filepath.Match(params.Include, fi.Name())
				if !matched {
					return nil
				}
			}

			// Skip binary/large files
			if fi.Size() > 10*1024*1024 { // 10MB
				return nil
			}

			fileMatches := searchFile(path, re, maxMatches-len(matches))
			matches = append(matches, fileMatches...)
			filesSearched++
			return nil
		})
	} else {
		// Search single file
		matches = searchFile(params.Path, re, maxMatches)
		filesSearched = 1
	}

	if len(matches) == 0 {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("No matches found for pattern %q", params.Pattern),
			ResultType:       "success",
			SessionLog:       fmt.Sprintf("grep: no matches for %q in %s", params.Pattern, params.Path),
		}, nil
	}

	// Format output
	var output strings.Builder
	if contextLines > 0 {
		// With context, we need to re-read files and show surrounding lines
		output.WriteString(formatMatchesWithContext(matches, contextLines))
	} else {
		for _, m := range matches {
			line := m.Line
			if len(line) > 500 {
				line = line[:500] + "..."
			}
			fmt.Fprintf(&output, "%s:%d: %s\n", m.File, m.LineNum, line)
		}
	}

	result := output.String()
	if len(matches) >= maxMatches {
		result += fmt.Sprintf("\n... (truncated at %d matches)", maxMatches)
	}

	return ToolResult{
		TextResultForLLM: result,
		ResultType:       "success",
		SessionLog:       fmt.Sprintf("grep: %d matches for %q in %s (%d files)", len(matches), params.Pattern, params.Path, filesSearched),
	}, nil
}

func searchFile(path string, re *regexp.Regexp, maxMatches int) []grepMatch {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var matches []grepMatch
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if len(matches) >= maxMatches {
			break
		}
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, grepMatch{
				File:    path,
				LineNum: lineNum,
				Line:    line,
			})
		}
	}

	return matches
}

func formatMatchesWithContext(matches []grepMatch, contextLines int) string {
	// Group matches by file
	fileMatches := make(map[string][]grepMatch)
	var fileOrder []string
	for _, m := range matches {
		if _, exists := fileMatches[m.File]; !exists {
			fileOrder = append(fileOrder, m.File)
		}
		fileMatches[m.File] = append(fileMatches[m.File], m)
	}

	var output strings.Builder
	for _, filePath := range fileOrder {
		fMatches := fileMatches[filePath]

		// Read the file lines
		lines, err := readFileLines(filePath)
		if err != nil {
			continue
		}

		fmt.Fprintf(&output, "--- %s ---\n", filePath)
		lastPrinted := -1

		for _, m := range fMatches {
			start := m.LineNum - contextLines - 1
			end := m.LineNum + contextLines - 1
			if start < 0 {
				start = 0
			}
			if end >= len(lines) {
				end = len(lines) - 1
			}

			if lastPrinted >= 0 && start > lastPrinted+1 {
				output.WriteString("  ...\n")
			}

			for i := start; i <= end; i++ {
				if i <= lastPrinted {
					continue
				}
				marker := " "
				if i == m.LineNum-1 {
					marker = ">"
				}
				line := lines[i]
				if len(line) > 500 {
					line = line[:500] + "..."
				}
				fmt.Fprintf(&output, "%s %4d: %s\n", marker, i+1, line)
				lastPrinted = i
			}
		}
		output.WriteString("\n")
	}

	return output.String()
}

func readFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
