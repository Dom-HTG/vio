package model

import (
	"context"

	"vio/internal/domain"
)

// Provider is implemented by every model backend. It is the port the agent
// talks to; concrete transports (OpenAI-compatible HTTP, fakes) live behind it.
type Provider interface {
	Chat(ctx context.Context, request Request) (Response, error)
}

type Request struct {
	Model    string
	Messages []domain.Message
}

type Response struct {
	Message       domain.Message
	ToolCalls     []domain.ToolCall // tool calls the model requested, if any
	UsageMetadata any
}
