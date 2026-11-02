package tools

import (
	"context"
	"fmt"
	"strings"

	"vio/internal/workspace"
)

type EditFile struct {
	ws workspace.FS
}

func NewEditFile(ws workspace.FS) *EditFile {
	return &EditFile{ws: ws}
}

func (t *EditFile) Name() string { return "edit_file" }

func (t *EditFile) Description() string {
	return "Replace one unique occurrence of text in an existing file."
}

func (t *EditFile) Schema() ToolSchema {
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
				"old_text": map[string]any{
					"type":        "string",
					"description": "Text to replace. Must appear exactly once.",
				},
				"new_text": map[string]any{
					"type":        "string",
					"description": "Replacement text.",
				},
			},
			"required": []string{"path", "old_text", "new_text"},
		},
	}
}

func (t *EditFile) Execute(_ context.Context, in ToolInput) (ToolOutput, error) {
	path, ok := argString(in, "path")
	if !ok || strings.TrimSpace(path) == "" {
		return failure("edit_file: missing required argument %q", "path"), nil
	}
	oldText, ok := argString(in, "old_text")
	if !ok || oldText == "" {
		return failure("edit_file: missing required argument %q", "old_text"), nil
	}
	newText, ok := argString(in, "new_text")
	if !ok {
		return failure("edit_file: missing required argument %q", "new_text"), nil
	}

	content, err := t.ws.Read(path)
	if err != nil {
		return failure("edit_file: %v", err), nil
	}

	text := string(content)
	switch count := strings.Count(text, oldText); {
	case count == 0:
		return failure("edit_file: old_text not found in %s", path), nil
	case count > 1:
		return failure("edit_file: old_text appears %d times in %s; include more context so it is unique", count, path), nil
	}

	updated := strings.Replace(text, oldText, newText, 1)
	if err := t.ws.Write(path, []byte(updated)); err != nil {
		return failure("edit_file: %v", err), nil
	}
	return ToolOutput{Output: fmt.Sprintf("edited %s", path)}, nil
}
