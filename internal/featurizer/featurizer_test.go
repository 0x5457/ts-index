package featurizer_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/0x5457/ts-index/internal/featurizer"
)

type fakeLLM struct{}

func (f *fakeLLM) Completion(
	ctx context.Context,
	req featurizer.CompletionRequest,
) (featurizer.CompletionResponse, error) {
	// Always return a deterministic tool call with all features set true
	args := map[string]bool{}
	if fun := req.Tools[0].Function; fun.Parameters != nil {
		if props, ok := fun.Parameters["properties"].(map[string]any); ok {
			for k := range props {
				args[k] = true
			}
		}
	}
	b, _ := json.Marshal(args)
	return featurizer.CompletionResponse{
		Choices: []featurizer.Choice{{
			Message: featurizer.ResponseMessage{
				ToolCalls: []featurizer.ToolCall{
					{
						Function: featurizer.FunctionCall{
							Name:      "extract_features",
							Arguments: string(b),
						},
					},
				},
			},
		}},
		Usage: featurizer.Usage{PromptTokens: 10, CompletionTokens: 5},
	}, nil
}

func Test_Featurizer_Embed_and_Batch(t *testing.T) {
	f := featurizer.Featurizer{
		SystemPrompt:  "you are a feature extractor",
		MessagePrefix: "please analyze",
		Features: []featurizer.Feature{
			{Identifier: "has_code", Description: "issue contains code"},
			{Identifier: "needs_debug", Description: "needs debugging"},
		},
		CreateLLM: func(cfg featurizer.LLMConfig) featurizer.LLM { return &fakeLLM{} },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	emb, err := f.Embed(ctx, "some description", featurizer.LLMConfig{Model: "fake"}, 0.7, 3)
	if err != nil {
		t.Fatalf("embed error: %v", err)
	}
	if len(emb.Samples) != 3 {
		t.Fatalf("expected 3 samples")
	}
	if coef, ok := emb.Coefficient("has_code"); !ok || coef != 1.0 {
		t.Fatalf("coef mismatch: %v %v", coef, ok)
	}

	issues := []string{"a", "b", "c"}
	embs, err := f.EmbedBatch(ctx, issues, featurizer.LLMConfig{Model: "fake"}, 0.7, 2)
	if err != nil {
		t.Fatalf("embed batch error: %v", err)
	}
	if len(embs) != len(issues) {
		t.Fatalf("result length mismatch")
	}
	for i, e := range embs {
		if len(e.Samples) != 2 {
			t.Fatalf("idx %d expected 2 samples", i)
		}
	}
}
