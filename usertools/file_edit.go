// File edit tool - performs find and replace in files
package usertools

import (
	"fmt"
	"os"
	"strings"
)

func init() {
	RegisterLazy(ToolDefinition{
		Name:        "file_edit",
		Description: "Edit a file by replacing exact text matches. Use this to make targeted changes to existing files without rewriting the entire file.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to edit",
				},
				"old_text": map[string]interface{}{
					"type":        "string",
					"description": "The exact text to find and replace",
				},
				"new_text": map[string]interface{}{
					"type":        "string",
					"description": "The replacement text",
				},
				"replace_all": map[string]interface{}{
					"type":        "boolean",
					"description": "Replace all occurrences (default: false, replaces first occurrence only)",
				},
			},
			"required": []string{"path", "old_text", "new_text"},
		},
		Loader: func() ToolHandler {
			return fileEditHandler
		},
	})
}

type fileEditParams struct {
	Path       string `json:"path"`
	OldText    string `json:"old_text"`
	NewText    string `json:"new_text"`
	ReplaceAll bool   `json:"replace_all"`
}

func fileEditHandler(invocation ToolInvocation) (ToolResult, error) {
	var params fileEditParams
	if err := MapToStruct(invocation.Arguments, &params); err != nil {
		return ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Path == "" {
		return ToolResult{
			TextResultForLLM: "Error: path parameter is required",
			ResultType:       "error",
		}, nil
	}
	if params.OldText == "" {
		return ToolResult{
			TextResultForLLM: "Error: old_text parameter is required",
			ResultType:       "error",
		}, nil
	}
	if params.OldText == params.NewText {
		return ToolResult{
			TextResultForLLM: "Error: old_text and new_text are identical",
			ResultType:       "error",
		}, nil
	}

	// Read the file
	data, err := os.ReadFile(params.Path)
	if err != nil {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error reading file: %v", err),
			ResultType:       "error",
			SessionLog:       fmt.Sprintf("file_edit read error: %v", err),
		}, nil
	}

	content := string(data)

	// Check if old_text exists
	count := strings.Count(content, params.OldText)
	if count == 0 {
		return ToolResult{
			TextResultForLLM: "Error: old_text not found in file",
			ResultType:       "error",
			SessionLog:       fmt.Sprintf("file_edit: text not found in %s", params.Path),
		}, nil
	}

	// For single replacement, ensure uniqueness to avoid ambiguity
	if !params.ReplaceAll && count > 1 {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error: old_text found %d times in file. Use replace_all=true to replace all, or provide more context to make old_text unique.", count),
			ResultType:       "error",
			SessionLog:       fmt.Sprintf("file_edit: ambiguous match (%d occurrences) in %s", count, params.Path),
		}, nil
	}

	// Perform replacement
	var newContent string
	replacements := 0
	if params.ReplaceAll {
		newContent = strings.ReplaceAll(content, params.OldText, params.NewText)
		replacements = count
	} else {
		newContent = strings.Replace(content, params.OldText, params.NewText, 1)
		replacements = 1
	}

	// Write back
	info, err := os.Stat(params.Path)
	if err != nil {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error getting file info: %v", err),
			ResultType:       "error",
		}, nil
	}

	if err := os.WriteFile(params.Path, []byte(newContent), info.Mode()); err != nil {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error writing file: %v", err),
			ResultType:       "error",
			SessionLog:       fmt.Sprintf("file_edit write error: %v", err),
		}, nil
	}

	return ToolResult{
		TextResultForLLM: fmt.Sprintf("Replaced %d occurrence(s) in %s", replacements, params.Path),
		ResultType:       "success",
		SessionLog:       fmt.Sprintf("file_edit: %d replacement(s) in %s", replacements, params.Path),
	}, nil
}
