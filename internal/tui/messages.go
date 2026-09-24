package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Role describes what kind of transcript entry a Message is.
type Role int

const (
	RoleUser Role = iota
	RoleAssistant
	RoleTool
	RoleError
	RoleInfo
)

// Message is a single entry in the chat transcript.
type Message struct {
	Role    Role
	Content string
}

// tokenMsg is an internal event carrying one chunk of streamed assistant text.
// It is the TUI-side equivalent of an agent EventModelToken delta.
type tokenMsg string

// streamDoneMsg signals the end of the current assistant stream.
type streamDoneMsg struct{}

// newTokenCmd returns a tea.Cmd that fires after d, delivering one token.
func newTokenCmd(token string, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return tokenMsg(token)
	})
}
