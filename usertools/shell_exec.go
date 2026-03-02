// Shell execution tool - runs shell commands and returns output
package usertools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func init() {
	RegisterLazy(ToolDefinition{
		Name:        "shell_exec",
		Description: "Execute a shell command and return its output. Supports PowerShell (default), cmd, and bash. Use this to run system commands, scripts, CLI tools, and automate tasks.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The command to execute",
				},
				"shell": map[string]interface{}{
					"type":        "string",
					"description": "Shell to use: powershell (default), cmd, or bash",
					"enum":        []string{"powershell", "cmd", "bash"},
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"description": "Timeout in seconds (default: 60, max: 300)",
				},
				"working_dir": map[string]interface{}{
					"type":        "string",
					"description": "Working directory for the command (defaults to current directory)",
				},
			},
			"required": []string{"command"},
		},
		Loader: func() ToolHandler {
			return shellExecHandler
		},
	})
}

type shellExecParams struct {
	Command    string  `json:"command"`
	Shell      string  `json:"shell"`
	Timeout    float64 `json:"timeout"`
	WorkingDir string  `json:"working_dir"`
}

func shellExecHandler(invocation ToolInvocation) (ToolResult, error) {
	var params shellExecParams
	if err := MapToStruct(invocation.Arguments, &params); err != nil {
		return ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Command == "" {
		return ToolResult{
			TextResultForLLM: "Error: command parameter is required",
			ResultType:       "error",
		}, nil
	}

	// Default shell
	if params.Shell == "" {
		params.Shell = "powershell"
	}

	// Default and cap timeout
	if params.Timeout <= 0 {
		params.Timeout = 60
	}
	if params.Timeout > 300 {
		params.Timeout = 300
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(params.Timeout)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	switch params.Shell {
	case "cmd":
		cmd = exec.CommandContext(ctx, "cmd", "/C", params.Command)
	case "bash":
		cmd = exec.CommandContext(ctx, "bash", "-c", params.Command)
	default: // powershell
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", params.Command)
	}

	if params.WorkingDir != "" {
		cmd.Dir = params.WorkingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	var output strings.Builder
	if stdout.Len() > 0 {
		output.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if output.Len() > 0 {
			output.WriteString("\n")
		}
		output.WriteString("STDERR:\n")
		output.WriteString(stderr.String())
	}

	// Truncate very long output
	result := output.String()
	const maxLen = 50000
	if len(result) > maxLen {
		result = result[:maxLen] + "\n... (output truncated)"
	}

	exitCode := 0
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ToolResult{
				TextResultForLLM: fmt.Sprintf("Command timed out after %.0f seconds.\n%s", params.Timeout, result),
				ResultType:       "error",
				SessionLog:       fmt.Sprintf("shell_exec timed out: %s", params.Command),
			}, nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	if exitCode != 0 {
		return ToolResult{
			TextResultForLLM: fmt.Sprintf("Command exited with code %d.\n%s", exitCode, result),
			ResultType:       "error",
			SessionLog:       fmt.Sprintf("shell_exec failed (exit %d): %s", exitCode, params.Command),
		}, nil
	}

	if result == "" {
		result = "(no output)"
	}

	return ToolResult{
		TextResultForLLM: result,
		ResultType:       "success",
		SessionLog:       fmt.Sprintf("shell_exec [%s]: %s", params.Shell, params.Command),
	}, nil
}
