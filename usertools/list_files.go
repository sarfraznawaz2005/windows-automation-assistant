// List files tool - lists directory contents and searches for files by glob pattern
package usertools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	RegisterLazy(ToolDefinition{
		Name:        "list_files",
		Description: "List files and directories. Can list a directory's contents or search for files matching a glob pattern (e.g. \"**/*.go\", \"*.txt\"). Use this to explore the filesystem and find files.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory path to list, or base path for glob search (default: current directory)",
				},
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Glob pattern to match files (e.g. \"*.go\", \"**/*.txt\"). If omitted, lists directory contents.",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "List files recursively (default: false). Ignored when pattern uses **.",
				},
			},
		},
		Loader: func() ToolHandler {
			return listFilesHandler
		},
	})
}

type listFilesParams struct {
	Path      string `json:"path"`
	Pattern   string `json:"pattern"`
	Recursive bool   `json:"recursive"`
}

func listFilesHandler(invocation ToolInvocation) (ToolResult, error) {
	var params listFilesParams
	if err := MapToStruct(invocation.Arguments, &params); err != nil {
		return ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Path == "" {
		params.Path = "."
	}

	// If pattern is provided, do glob matching
	if params.Pattern != "" {
		return globSearch(params.Path, params.Pattern)
	}

	// Otherwise list directory contents
	return listDirectory(params.Path, params.Recursive)
}

func listDirectory(dir string, recursive bool) (ToolResult, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error: %v", err),
			ResultType:       "error",
		}, nil
	}
	if !info.IsDir() {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Error: %s is not a directory", dir),
			ResultType:       "error",
		}, nil
	}

	var output strings.Builder
	count := 0
	const maxFiles = 1000

	if recursive {
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip errors
			}
			if count >= maxFiles {
				return filepath.SkipAll
			}
			// Skip hidden directories
			if info.IsDir() && strings.HasPrefix(info.Name(), ".") && path != dir {
				return filepath.SkipDir
			}
			rel, _ := filepath.Rel(dir, path)
			if rel == "." {
				return nil
			}
			prefix := "  "
			if info.IsDir() {
				prefix = "D "
			}
			fmt.Fprintf(&output, "%s%s\n", prefix, rel)
			count++
			return nil
		})
	} else {
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			return ToolResult{
				TextResultForLLM: fmt.Sprintf("Error reading directory: %v", readErr),
				ResultType:       "error",
			}, nil
		}
		for _, entry := range entries {
			if count >= maxFiles {
				break
			}
			prefix := "  "
			if entry.IsDir() {
				prefix = "D "
			}
			fmt.Fprintf(&output, "%s%s\n", prefix, entry.Name())
			count++
		}
	}

	result := output.String()
	if count >= maxFiles {
		result += fmt.Sprintf("\n... (truncated at %d entries)", maxFiles)
	}
	if count == 0 {
		result = "(empty directory)"
	}

	return ToolResult{
		TextResultForLLM: result,
		ResultType:       "success",
		SessionLog:       fmt.Sprintf("list_files: %s (%d entries)", dir, count),
	}, nil
}

func globSearch(basePath, pattern string) (ToolResult, error) {
	var matches []string
	const maxMatches = 1000

	// Handle ** pattern (recursive glob)
	if strings.Contains(pattern, "**") {
		err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if len(matches) >= maxMatches {
				return filepath.SkipAll
			}
			// Skip hidden directories
			if info.IsDir() && strings.HasPrefix(info.Name(), ".") && path != basePath {
				return filepath.SkipDir
			}

			rel, _ := filepath.Rel(basePath, path)
			// Extract the non-** part of the pattern for matching
			simplePattern := strings.ReplaceAll(pattern, "**/", "")
			matched, _ := filepath.Match(simplePattern, info.Name())
			if matched {
				matches = append(matches, rel)
			}
			return nil
		})
		if err != nil {
			return ToolResult{
				TextResultForLLM: fmt.Sprintf("Error during search: %v", err),
				ResultType:       "error",
			}, nil
		}
	} else {
		// Simple glob
		fullPattern := filepath.Join(basePath, pattern)
		var err error
		matches, err = filepath.Glob(fullPattern)
		if err != nil {
			return ToolResult{
				TextResultForLLM: fmt.Sprintf("Error: invalid glob pattern: %v", err),
				ResultType:       "error",
			}, nil
		}
		// Convert to relative paths
		for i, m := range matches {
			if rel, err := filepath.Rel(basePath, m); err == nil {
				matches[i] = rel
			}
		}
	}

	if len(matches) == 0 {
		return ToolResult{
			TextResultForLLM: "No files matched the pattern.",
			ResultType:       "success",
			SessionLog:       fmt.Sprintf("list_files glob: %s (0 matches)", pattern),
		}, nil
	}

	var output strings.Builder
	for _, m := range matches {
		fmt.Fprintf(&output, "%s\n", m)
	}

	result := output.String()
	if len(matches) >= maxMatches {
		result += fmt.Sprintf("\n... (truncated at %d matches)", maxMatches)
	}

	return ToolResult{
		TextResultForLLM: result,
		ResultType:       "success",
		SessionLog:       fmt.Sprintf("list_files glob: %s (%d matches)", pattern, len(matches)),
	}, nil
}
