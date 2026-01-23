// Sum tool - adds two numbers together
// This is an example tool demonstrating the lazy loading usertools architecture.
package usertools

import (
	"fmt"
)

func init() {
	RegisterLazy(ToolDefinition{
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
		Loader: func() ToolHandler {
			// No expensive initialization needed for this simple tool
			return sumHandler
		},
	})
}

// sumParams defines the parameters for the sum tool
type sumParams struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

// sumHandler implements the sum tool functionality
func sumHandler(invocation ToolInvocation) (ToolResult, error) {
	// Parse parameters
	var params sumParams
	if err := MapToStruct(invocation.Arguments, &params); err != nil {
		return ToolResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	// Perform the calculation
	result := params.A + params.B

	// Format the result for the LLM
	textResult := fmt.Sprintf("The sum of %v and %v is %v", params.A, params.B, result)

	return ToolResult{
		TextResultForLLM: textResult,
		ResultType:       "success",
		SessionLog:       fmt.Sprintf("Calculated: %v + %v = %v", params.A, params.B, result),
	}, nil
}
