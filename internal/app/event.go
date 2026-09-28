package app

type EventType string

// Events emitted by the agent runtime.
// These events will be consumed by the TUI.
const (
	EventAgentStarted  EventType = "agent_started"
	EventModelStarted  EventType = "model_started"
	EventModelOutput   EventType = "model_output"
	EventToolStarted   EventType = "tool_started"
	EventToolOutput    EventType = "tool_output"
	EventToolFinished  EventType = "tool_finished"
	EventAgentFinished EventType = "agent_finished"
	EventError         EventType = "error"
)

// Event is a single message on the app's internal event stream. The agent
// runtime is the sole producer and the TUI is the sole consumer; Data carries
// the type-specific payload defined alongside each EventType constant.
type Event struct {
	Type EventType
	Data any
}

// AgentStarted signals the start of a turn for a user prompt.
type AgentStarted struct {
	Prompt string
}

// ModelStarted signals that a request has been sent to the model provider.
type ModelStarted struct {
	Model string
}

// ModelOutput carries one streamed chunk of assistant text.
type ModelOutput struct {
	Text string
}

// ToolStarted signals that the runtime is about to execute a tool call.
type ToolStarted struct {
	ID   string
	Name string
	Args map[string]any
}

// ToolOutput carries one streamed chunk of output from a running tool.
type ToolOutput struct {
	ID   string
	Text string
}

// ToolFinished signals that a tool call completed, successfully or not.
type ToolFinished struct {
	ID        string
	Name      string
	Ok        bool
	Truncated bool
}

// AgentFinished signals the end of a turn and carries the final assistant text.
type AgentFinished struct {
	Text string
}

// AgentError signals a turn-terminating failure.
type AgentError struct {
	Err error
}
