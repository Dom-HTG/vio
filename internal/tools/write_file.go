package tools

import (
	"context"
	"fmt"
	"strings"

	"vio/internal/workspace"
)

type WriteFile struct {
	ws workspace.FS
}

func NewWriteFile(ws workspace.FS) *WriteFile {
	return &WriteFile{ws: ws}
}

func (t *WriteFile) Name() string { return "write_file" }

func (t *WriteFile) Description() string {
	return "Create or replace a file in the repository."
}

func (t *WriteFile) Schema() ToolSchema {
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
				"content": map[string]any{
					"type":        "string",
					"description": "Full file contents.",
				},
			},
			"required": []string{"path", "content"},
		},
	}
}

func (t *WriteFile) Execute(_ context.Context, in ToolInput) (ToolOutput, error) {
	path, ok := argString(in, "path")
	if !ok || strings.TrimSpace(path) == "" {
		return failure("write_file: missing required argument %q", "path"), nil
	}
	content, ok := argString(in, "content")
	if !ok {
		return failure("write_file: missing required argument %q", "content"), nil
	}

	if err := t.ws.Write(path, []byte(content)); err != nil {
		return failure("write_file: %v", err), nil
	}
	return ToolOutput{Output: fmt.Sprintf("wrote %d bytes to %s", len(content), path)}, nil
}
