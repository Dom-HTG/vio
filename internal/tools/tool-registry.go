package tools

import (
	"fmt"
	"sort"

	"vio/internal/domain"
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("tools: cannot register a nil tool")
	}
	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tools: tool has no name")
	}
	if r.tools == nil {
		r.tools = make(map[string]Tool)
	}
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tools: %q is already registered", name)
	}
	r.tools[name] = tool
	return nil
}

func (r *Registry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *Registry) Definitions() []domain.ToolDefinition {
	names := r.Names()
	defs := make([]domain.ToolDefinition, 0, len(names))
	for _, name := range names {
		schema := r.tools[name].Schema()
		defs = append(defs, domain.ToolDefinition{
			Name:        name,
			Description: schema.Description,
			Params:      schema.Params,
		})
	}
	return defs
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
