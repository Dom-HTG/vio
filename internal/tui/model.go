package tui

import (
	"unicode/utf8"

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
		case "ctrl+c":
			return m, tea.Quit

		// handle enter key press.
		case "enter":
			// handle enter key press here.
			return m, nil

		// handle backspace key press.
		case "backspace":
			if len(m.input) > 0 {
				_, size := utf8.DecodeLastRuneInString(m.input)
				m.input = m.input[:len(m.input)-size]
			}

		default:
			// only append printable characters; ignore special keys.
			if text := msg.Key().Text; text != "" {
				m.input += text
			}
		}
	}
	return m, nil
}

// render update to the terminal UI.
func (m *model) View() tea.View {
	return tea.NewView(header + "\n\n> " + m.input + "\n")
}
