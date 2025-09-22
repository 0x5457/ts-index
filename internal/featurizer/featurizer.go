package featurizer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// systemMessage and userMessage helpers
func (f Featurizer) systemMessage() Message {
	return Message{Role: "system", Content: f.SystemPrompt}
}

func (f Featurizer) userMessage(issue string) Message {
	if f.MessagePrefix != "" {
		return Message{Role: "user", Content: f.MessagePrefix + "\n\n" + issue}
	}
	return Message{Role: "user", Content: issue}
}

// toolDescription builds the JSON schema for boolean features
func (f Featurizer) toolDescription() Tool {
	props := map[string]any{}
	required := []string{}
	for _, ft := range f.Features {
		props[ft.Identifier] = ft.ToToolDescriptionField()
		required = append(required, ft.Identifier)
	}
	params := map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "extract_features",
			Description: "Return boolean evaluations for defined features.",
			Parameters:  params,
		},
	}
}

func (f Featurizer) toolChoice() *ToolChoice {
	tc := &ToolChoice{Type: "function"}
	tc.Function.Name = "extract_features"
	return tc
}

// Embed extracts features for a single issue with multiple samples to reduce variance.
func (f Featurizer) Embed(
	ctx context.Context,
	issue string,
	cfg LLMConfig,
	temperature float64,
	samples int,
) (FeatureEmbedding, error) {
	if f.CreateLLM == nil {
		return FeatureEmbedding{}, errors.New("CreateLLM is nil")
	}
	if samples <= 0 {
		samples = 1
	}
	llm := f.CreateLLM(cfg)

	var out FeatureEmbedding
	out.Samples = make([]map[string]bool, 0, samples)
	var latencySum float64
	var promptSum, completionSum int

	req := CompletionRequest{
		Messages:    []Message{f.systemMessage(), f.userMessage(issue)},
		Tools:       []Tool{f.toolDescription()},
		ToolChoice:  f.toolChoice(),
		Temperature: temperature,
		Model:       cfg.Model,
	}

	for i := 0; i < samples; i++ {
		start := time.Now()
		resp, err := llm.Completion(ctx, req)
		if err != nil {
			return FeatureEmbedding{}, err
		}
		elapsed := time.Since(start).Seconds()
		embedding, err := parseToolArgs(resp)
		if err != nil {
			return FeatureEmbedding{}, err
		}
		out.Samples = append(out.Samples, embedding)
		latencySum += elapsed
		promptSum += resp.Usage.PromptTokens
		completionSum += resp.Usage.CompletionTokens
	}

	out.ResponseLatency = ptrf(latencySum)
	out.PromptTokens = ptri(promptSum)
	out.CompletionTokens = ptri(completionSum)
	return out, nil
}

// EmbedBatch runs Embed concurrently for a batch and preserves input order.
func (f Featurizer) EmbedBatch(
	ctx context.Context,
	issues []string,
	cfg LLMConfig,
	temperature float64,
	samples int,
) ([]FeatureEmbedding, error) {
	type result struct {
		idx int
		emb FeatureEmbedding
		err error
	}
	out := make([]FeatureEmbedding, len(issues))
	ch := make(chan result, len(issues))
	var wg sync.WaitGroup
	for i, desc := range issues {
		wg.Add(1)
		go func(idx int, d string) {
			defer wg.Done()
			emb, err := f.Embed(ctx, d, cfg, temperature, samples)
			ch <- result{idx: idx, emb: emb, err: err}
		}(i, desc)
	}
	go func() { wg.Wait(); close(ch) }()

	var firstErr error
	for r := range ch {
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
		out[r.idx] = r.emb
	}
	return out, firstErr
}

func parseToolArgs(resp CompletionResponse) (map[string]bool, error) {
	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		return nil, fmt.Errorf("no tool call in response")
	}
	args := resp.Choices[0].Message.ToolCalls[0].Function.Arguments
	// tolerate either proper JSON object or stringified JSON
	var m map[string]bool
	if err := json.Unmarshal([]byte(args), &m); err == nil {
		return m, nil
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(args), &raw); err != nil {
		return nil, err
	}
	m = make(map[string]bool, len(raw))
	for k, v := range raw {
		switch vv := v.(type) {
		case bool:
			m[k] = vv
		case float64:
			m[k] = vv != 0
		case string:
			m[k] = vv == "true"
		}
	}
	return m, nil
}

func ptri(v int) *int         { return &v }
func ptrf(v float64) *float64 { return &v }
