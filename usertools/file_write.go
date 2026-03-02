// File write tool - creates or overwrites files
package usertools

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	RegisterLazy(ToolDefinition{
		Name:        "file_write",
		Description: "Write content to a file. Creates the file and any parent directories if they don't exist. By default overwrites existing files; set append to true to append instead.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to write to",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The content to write",
				},
				"append": map[string]interface{}{
					"type":        "boolean",
					"description": "Append to file instead of overwriting (default: false)",
				},
			},
			"required": []string{"path", "content"},
		},
		Loader: func() ToolHandler {
			return fileWriteHandler
		},
	})
}

type fileWriteParams struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Append  bool   `json:"append"`
}

func fileWriteHandler(invocation ToolInvocation) (ToolResult, error) {
	var params fileWriteParams
	if err := MapToStruct(invocation.Arguments, &params); err != nil {
		return ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Path == "" {
		return ToolResult{
			TextResultForLLM: "Error: path parameter is required",
			ResultType:       "error",
		}, nil
	}

	// Create parent directories if needed
	dir := filepath.Dir(params.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return ToolResult{
				TextResultForLLM: fmt.Sprintf("Error creating directories: %v", err),
				ResultType:       "error",
				SessionLog:       fmt.Sprintf("file_write mkdir error: %v", err),
			}, nil
		}
	}

	var flag int
	if params.Append {
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	} else {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	file, err := os.OpenFile(params.Path, flag, 0644)
	if err != nil {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error opening file: %v", err),
			ResultType:       "error",
			SessionLog:       fmt.Sprintf("file_write error: %v", err),
		}, nil
	}
	defer file.Close()

	n, err := file.WriteString(params.Content)
	if err != nil {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error writing file: %v", err),
			ResultType:       "error",
			SessionLog:       fmt.Sprintf("file_write error: %v", err),
		}, nil
	}

	action := "Written"
	if params.Append {
		action = "Appended"
	}

	return ToolResult{
		TextResultForLLM: fmt.Sprintf("%s %d bytes to %s", action, n, params.Path),
		ResultType:       "success",
		SessionLog:       fmt.Sprintf("file_write: %s %d bytes to %s", action, n, params.Path),
	}, nil
}
