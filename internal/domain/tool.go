package domain

// ToolDefinition is the data-only metadata describing a tool to the model: its
// name, description, and JSON Schema for its parameters. It deliberately omits
// execution so the provider layer can translate it to a wire format without
// ever being able to run a tool.
type ToolDefinition struct {
	Name        string
	Description string
	Params      map[string]any
}
