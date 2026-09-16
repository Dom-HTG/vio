package tui

import (
	tea "charm.land/bubbletea/v2"
)

// type ModelContract interface {
// 	Init() tea.Cmd
// 	Update(msg tea.Msg) (tea.Model, tea.Cmd)
// 	View() string
// }

type model struct {
	input    string
	messages []Message
}

type Message struct {
	Role    string
	Content string
}

func NewModel() *model {
	return &model{}
}

// start the terminal UI.
// bubbletea calls init when the application starts.
func (m *model) Init() tea.Cmd {
	return nil
}

// update application state.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// handle quit messages.
	case tea.KeyMsg:
		switch msg.String() {

		// handle quit messages.
		case "ctrl+c", "q":
			return m, tea.Quit

		// handle enter key press.
		case "enter":
			// handle enter key press here.
			return m, nil

		// handle backspace key press.
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}

		default:
			m.input += msg.String()
		}
	}
	return m, nil
}

// render update to the terminal UI.
func (m *model) View() string {
	return "> " + m.input + "\n"
}
