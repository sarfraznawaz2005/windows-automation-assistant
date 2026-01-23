// Notepad tool - opens the Windows Notepad application
package usertools

import (
	"fmt"
	"os/exec"
)

func init() {
	RegisterLazy(ToolDefinition{
		Name:        "open_notepad",
		Description: "Opens the Windows Notepad application",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Loader: func() ToolHandler {
			return notepadHandler
		},
	})
}

// notepadHandler opens Notepad
func notepadHandler(invocation ToolInvocation) (ToolResult, error) {
	cmd := exec.Command("notepad.exe")
	err := cmd.Start()
	if err != nil {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Failed to open Notepad: %v", err),
			ResultType:       "error",
			SessionLog:       fmt.Sprintf("Notepad launch error: %v", err),
		}, nil
	}

	return ToolResult{
		TextResultForLLM: "Notepad has been opened successfully.",
		ResultType:       "success",
		SessionLog:       "Opened Notepad application",
	}, nil
}
