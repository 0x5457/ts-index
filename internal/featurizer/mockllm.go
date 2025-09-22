package featurizer

import (
	"context"
	"encoding/json"
)

// MockLLM returns a deterministic tool call with all features defaulting to false
// This allows running the pipeline without external LLM access.
type MockLLM struct{}

func (m *MockLLM) Completion(
	ctx context.Context,
	req CompletionRequest,
) (CompletionResponse, error) {
	args := map[string]bool{}
	if len(req.Tools) > 0 {
		if props, ok := req.Tools[0].Function.Parameters["properties"].(map[string]any); ok {
			for k := range props {
				// leave default false or set via simple heuristic if desired
				_ = k
			}
		}
	}
	b, _ := json.Marshal(args)
	return CompletionResponse{
		Choices: []Choice{{
			Message: ResponseMessage{
				ToolCalls: []ToolCall{
					{Function: FunctionCall{Name: "extract_features", Arguments: string(b)}},
				},
			},
		}},
	}, nil
}
