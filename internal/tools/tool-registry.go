package tools

type Registry struct {
	tools map[string]Tool
}

func (r *Registry) Register(tool Tool) error // error on duplicate name
func (r *Registry) Get(name string) (Tool, bool)
func (r *Registry) Definitions() []ToolDefinition // metadata for the provider
func (r *Registry) Names() []string
