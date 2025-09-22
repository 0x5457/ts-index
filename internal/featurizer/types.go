package featurizer

import (
	"context"
	"time"
)

// Feature represents a single boolean feature that can be extracted by an LLM
type Feature struct {
	Identifier  string
	Description string
}

// ToToolDescriptionField converts this feature into a JSON schema field
func (f Feature) ToToolDescriptionField() map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": f.Description,
	}
}

// FeatureEmbedding holds sampled feature evaluations and usage metadata
type FeatureEmbedding struct {
	Samples          []map[string]bool
	PromptTokens     *int
	CompletionTokens *int
	ResponseLatency  *float64
}

// Coefficient returns the average coefficient (0.0-1.0) for the given feature identifier.
// The second return value indicates whether the dimension was present in any sample.
func (e FeatureEmbedding) Coefficient(dimension string) (float64, bool) {
	var sum float64
	var count int
	for _, s := range e.Samples {
		if v, ok := s[dimension]; ok {
			if v {
				sum += 1
			}
			count++
		}
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}

// Featurizer orchestrates LLM tool-calling based extraction
type Featurizer struct {
	SystemPrompt  string
	MessagePrefix string
	Features      []Feature
	// CreateLLM constructs an LLM client from the provided config
	CreateLLM func(LLMConfig) LLM
}

// LLMConfig contains provider configuration
type LLMConfig struct {
	Model   string
	APIKey  string
	BaseURL string
	Timeout time.Duration
}

// LLM abstracts a tool-calling capable chat completion API
type LLM interface {
	Completion(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

// Message represents a chat message
type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// Tool and tool choice schema (OpenAI-style)
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolChoice struct {
	Type     string `json:"type"`
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

// CompletionRequest holds chat/tool-calling inputs
type CompletionRequest struct {
	Messages    []Message
	Tools       []Tool
	ToolChoice  *ToolChoice
	Temperature float64
	Model       string
}

// CompletionResponse is a simplified OpenAI-like response
type CompletionResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Message ResponseMessage `json:"message"`
}

type ResponseMessage struct {
	ToolCalls []ToolCall `json:"tool_calls"`
}

type ToolCall struct {
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // raw JSON string
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}
