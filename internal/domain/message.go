// Package domain holds the provider-independent core types shared across the
// agent, model, state, and tools packages.
//
// Nothing here may import another internal package, a UI toolkit, or a vendor
// SDK: these are the innermost types, and every other package depends on them
// rather than the other way around.
package domain

// Role identifies the author of a Message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is one turn in a conversation, independent of any provider.
type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

// ToolCall is a model-requested invocation of a tool. ID correlates the call
// with the tool-result message that answers it.
type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}
