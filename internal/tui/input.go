package tui

import "charm.land/bubbles/v2/textinput"

// newInput builds the text input component used to capture the user's prompt.
func newInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "describe a task for the agent…"
	ti.Prompt = "❯ "
	ti.Focus()
	return ti
}
