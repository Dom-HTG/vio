package tools

import "context"

type Tool interface {
	Name() string
	Description() string
	Schema() ToolSchema
	Execute(context.Context, ToolInput) (ToolOutput, error)
}

// JSON-Schema object for the tool's input parameters.
// The schema is used to validate the input arguments before executing the tool.
type ToolSchema struct {
	Name        string
	Description string
	Params      map[string]any // JSON Schema object
}

type ToolInput struct {
	Args map[string]any
}

type ToolOutput struct {
	Output  string
	IsError bool
	Data    map[string]any // optional structured payload for the UI
}
