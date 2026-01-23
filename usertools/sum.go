// Sum tool - adds two numbers together
// This is an example tool demonstrating the usertools architecture.
package usertools

import (
	"fmt"

	copilot "github.com/github/copilot-sdk/go"
)

func init() {
	Register(Tool{
		Name:        "sum",
		Description: "Adds two numbers together and returns the result",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": map[string]interface{}{
					"type":        "number",
					"description": "The first number",
				},
				"b": map[string]interface{}{
					"type":        "number",
					"description": "The second number",
				},
			},
			"required": []string{"a", "b"},
		},
		Handler: sumHandler,
	})
}

// sumParams defines the parameters for the sum tool
type sumParams struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

// sumHandler implements the sum tool functionality
func sumHandler(invocation copilot.ToolInvocation) (copilot.ToolResult, error) {
	// Parse parameters
	var params sumParams
	if err := MapToStruct(invocation.Arguments, &params); err != nil {
		return copilot.ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	// Perform the calculation
	result := params.A + params.B

	// Format the result for the LLM
	textResult := fmt.Sprintf("The sum of %v and %v is %v", params.A, params.B, result)

	return copilot.ToolResult{
		TextResultForLLM: textResult,
		ResultType:       "success",
		SessionLog:       fmt.Sprintf("Calculated: %v + %v = %v", params.A, params.B, result),
	}, nil
}
