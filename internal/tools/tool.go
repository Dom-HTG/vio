package tools

import (
	"context"
	"fmt"
)

type Tool interface {
	Name() string
	Description() string
	Schema() ToolSchema
	Execute(context.Context, ToolInput) (ToolOutput, error)
}

type ToolSchema struct {
	Name        string
	Description string
	Params      map[string]any
}

type ToolInput struct {
	Args map[string]any
}

type ToolOutput struct {
	Output  string
	IsError bool
	Data    map[string]any
}

func failure(format string, args ...any) ToolOutput {
	return ToolOutput{Output: fmt.Sprintf(format, args...), IsError: true}
}

func argString(in ToolInput, name string) (string, bool) {
	s, ok := in.Args[name].(string)
	return s, ok
}

func argBool(in ToolInput, name string) (bool, bool) {
	b, ok := in.Args[name].(bool)
	return b, ok
}

func argNumber(in ToolInput, name string) (float64, bool) {
	switch v := in.Args[name].(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}
