package model

import "context"

// every model implements the provider interface.
type Provider interface {
	Chat(ctx context.Context, request Request) (Response, error)
}

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

type ToolCall struct {
	Name string
	Args map[string]any
}

type Request struct {
	Model    string
	Messages []Message
}

type Response struct {
	Message       Message
	ToolCall      *ToolCall // optional tool call request from the model
	UsageMetadata any
}
