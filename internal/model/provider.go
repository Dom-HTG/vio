package model

import (
	"context"

	"vio/internal/domain"
)

type Provider interface {
	Generate(ctx context.Context, msgs []domain.Message, tools []domain.ToolDefinition) (*Response, error)
	Stream(ctx context.Context, msgs []domain.Message, tools []domain.ToolDefinition) (<-chan StreamEvent, error)
}

// StreamEventType is implemented for the provider-specific events.
type StreamEventType string

const (
	StreamToken    StreamEventType = "token"           // Text carries a text delta
	StreamToolCall StreamEventType = "tool_call_delta" // ToolCall carries a partial call
	StreamDone     StreamEventType = "done"            // completion
	StreamError    StreamEventType = "error"           // Error carries the failure
)

type StreamEvent struct {
	Type     StreamEventType
	Text     string
	ToolCall *domain.ToolCall
	Error    error
}

// type Request struct {
// 	Model    string
// 	Messages []domain.Message
// }

type Response struct {
	Message       domain.Message
	ToolCalls     []domain.ToolCall
	FinishReason  string
	UsageMetadata any
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
