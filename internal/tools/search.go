package tools

import (
	"context"
	"fmt"
	"strings"

	"vio/internal/workspace"
)

type Search struct {
	ws workspace.FS
}

func NewSearch(ws workspace.FS) *Search {
	return &Search{ws: ws}
}

func (t *Search) Name() string { return "search" }

func (t *Search) Description() string {
	return "Search repository files for a text query."
}

func (t *Search) Schema() ToolSchema {
	return ToolSchema{
		Name:        t.Name(),
		Description: t.Description(),
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Text to search for.",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Optional file or directory to search, relative to the repository root.",
				},
				"case_sensitive": map[string]any{
					"type":        "boolean",
					"description": "Match case exactly. Defaults to false.",
				},
			},
			"required": []string{"query"},
		},
	}
}

func (t *Search) Execute(_ context.Context, in ToolInput) (ToolOutput, error) {
	query, ok := argString(in, "query")
	if !ok || strings.TrimSpace(query) == "" {
		return failure("search: missing required argument %q", "query"), nil
	}

	opts := workspace.SearchOptions{}
	if path, ok := argString(in, "path"); ok {
		opts.Path = path
	}
	if caseSensitive, ok := argBool(in, "case_sensitive"); ok {
		opts.CaseSensitive = caseSensitive
	}

	matches, err := t.ws.Search(query, opts)
	if err != nil {
		return failure("search: %v", err), nil
	}
	if len(matches) == 0 {
		return ToolOutput{Output: fmt.Sprintf("no matches for %q", query)}, nil
	}

	var out strings.Builder
	for _, match := range matches {
		fmt.Fprintf(&out, "%s:%d: %s\n", match.Path, match.Line, match.Text)
	}
	return ToolOutput{Output: strings.TrimRight(out.String(), "\n")}, nil
}
