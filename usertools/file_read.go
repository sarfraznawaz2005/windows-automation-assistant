// File read tool - reads file contents
package usertools

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func init() {
	RegisterLazy(ToolDefinition{
		Name:        "file_read",
		Description: "Read the contents of a file. Supports reading entire files or specific line ranges. Returns the file content with line numbers.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to read",
				},
				"offset": map[string]interface{}{
					"type":        "number",
					"description": "Starting line number (1-based, default: 1)",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of lines to read (default: 2000)",
				},
			},
			"required": []string{"path"},
		},
		Loader: func() ToolHandler {
			return fileReadHandler
		},
	})
}

type fileReadParams struct {
	Path   string  `json:"path"`
	Offset float64 `json:"offset"`
	Limit  float64 `json:"limit"`
}

func fileReadHandler(invocation ToolInvocation) (ToolResult, error) {
	var params fileReadParams
	if err := MapToStruct(invocation.Arguments, &params); err != nil {
		return ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Path == "" {
		return ToolResult{
			TextResultForLLM: "Error: path parameter is required",
			ResultType:       "error",
		}, nil
	}

	offset := int(params.Offset)
	if offset < 1 {
		offset = 1
	}

	limit := int(params.Limit)
	if limit <= 0 {
		limit = 2000
	}

	file, err := os.Open(params.Path)
	if err != nil {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error reading file: %v", err),
			ResultType:       "error",
			SessionLog:       fmt.Sprintf("file_read error: %v", err),
		}, nil
	}
	defer file.Close()

	var output strings.Builder
	scanner := bufio.NewScanner(file)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	linesRead := 0
	totalLines := 0

	for scanner.Scan() {
		lineNum++
		totalLines = lineNum

		if lineNum < offset {
			continue
		}
		if linesRead >= limit {
			continue // keep counting total lines
		}

		line := scanner.Text()
		// Truncate very long lines
		if len(line) > 2000 {
			line = line[:2000] + "... (truncated)"
		}

		fmt.Fprintf(&output, "%6d\t%s\n", lineNum, line)
		linesRead++
	}

	if err := scanner.Err(); err != nil {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error reading file: %v", err),
			ResultType:       "error",
			SessionLog:       fmt.Sprintf("file_read scan error: %v", err),
		}, nil
	}

	if linesRead == 0 {
		if totalLines == 0 {
			return ToolResult{
				TextResultForLLM: fmt.Sprintf("File %s is empty.", params.Path),
				ResultType:       "success",
				SessionLog:       fmt.Sprintf("file_read: %s (empty)", params.Path),
			}, nil
		}
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("No lines in range. File has %d lines total.", totalLines),
			ResultType:       "success",
			SessionLog:       fmt.Sprintf("file_read: %s (offset beyond end)", params.Path),
		}, nil
	}

	result := output.String()
	if linesRead < totalLines-offset+1 {
		result += fmt.Sprintf("\n... (%d more lines, %d total)", totalLines-offset-linesRead+1, totalLines)
	}

	return ToolResult{
		TextResultForLLM: result,
		ResultType:       "success",
		SessionLog:       fmt.Sprintf("file_read: %s (lines %d-%d of %d)", params.Path, offset, offset+linesRead-1, totalLines),
	}, nil
}
