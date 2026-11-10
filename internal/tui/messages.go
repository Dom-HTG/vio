package tui

import (
	tea "charm.land/bubbletea/v2"

	"vio/internal/app"
)

// Role describes what kind of transcript entry a Message is.
type Role int

const (
	RoleUser Role = iota
	RoleAssistant
	RoleTool
	RoleToolResult
	RoleError
	RoleInfo
)

// Message is a single entry in the chat transcript.
type Message struct {
	Role    Role
	Content string
}

// eventMsg carries one agent event into the Bubble Tea update loop.
type eventMsg struct {
	Event app.Event
}

// streamDoneMsg signals that the agent's event channel closed.
type streamDoneMsg struct{}

// waitForEvent blocks until the next agent event or the stream closes.
func waitForEvent(events <-chan app.Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return streamDoneMsg{}
		}
		return eventMsg{Event: event}
	}
}
