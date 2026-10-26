package tools

import (
	"context"
	"strings"

	"vio/internal/workspace"
)

type ReadFile struct {
	ws workspace.FS
}

func NewReadFile(ws workspace.FS) *ReadFile {
	return &ReadFile{ws: ws}
}

func (t *ReadFile) Name() string { return "read_file" }

func (t *ReadFile) Description() string {
	return "Read a text file from the repository."
}

func (t *ReadFile) Schema() ToolSchema {
	return ToolSchema{
		Name:        t.Name(),
		Description: t.Description(),
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path relative to the repository root.",
				},
			},
			"required": []string{"path"},
		},
	}
}

func (t *ReadFile) Execute(_ context.Context, in ToolInput) (ToolOutput, error) {
	path, ok := argString(in, "path")
	if !ok || strings.TrimSpace(path) == "" {
		return failure("read_file: missing required argument %q", "path"), nil
	}

	content, err := t.ws.Read(path)
	if err != nil {
		return failure("read_file: %v", err), nil
	}
	return ToolOutput{Output: string(content)}, nil
}
