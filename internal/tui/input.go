package tui

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// newInput builds the text input component used to capture the user's prompt.
func newInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "describe a task for the agent…"
	ti.Prompt = "❯ "
	ti.Focus()
	return ti
}

// runPrompt is the seam where the agent runtime will be wired in. For now it
// streams a placeholder reply so the full TUI loop (submit → stream → done)
// can be exercised without a model provider.
//
// TODO: replace with a tea.Cmd that starts an agent turn and forwards
// agent.Events as tea.Msg values.
func runPrompt(text string) tea.Cmd {
	reply := "placeholder response — the agent loop will be wired in here. you said: " + text
	words := strings.Split(reply, " ")
	cmds := make([]tea.Cmd, 0, len(words)+1)
	for _, w := range words {
		cmds = append(cmds, newTokenCmd(w+" ", 30*time.Millisecond))
	}
	cmds = append(cmds, tea.Tick(30*time.Millisecond, func(time.Time) tea.Msg {
		return streamDoneMsg{}
	}))
	return tea.Sequence(cmds...)
}
