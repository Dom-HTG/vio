package tui

import (
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	gloss "charm.land/lipgloss/v2"
)

type model struct {
	width  int
	height int
	ready  bool

	input    textinput.Model
	viewport viewport.Model

	messages []Message
	busy     bool

	modelName string
	cwd       string
}

func NewModel() *model {
	vp := viewport.New()
	vp.SoftWrap = true

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}
	modelName := os.Getenv("MODEL_NAME")
	if modelName == "" {
		modelName = "unset"
	}

	return &model{
		input:     newInput(),
		viewport:  vp,
		modelName: modelName,
		cwd:       cwd,
	}
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.resize()
		return m, nil

	case tea.KeyPressMsg:
		return m, m.handleKey(msg)

	case tea.PasteMsg:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tokenMsg:
		if n := len(m.messages); n > 0 && m.messages[n-1].Role == RoleAssistant {
			m.messages[n-1].Content += string(msg)
			m.refreshViewport()
		}
		return m, nil

	case streamDoneMsg:
		m.busy = false
		return m, nil
	}

	return m, nil
}

func (m *model) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		return tea.Quit
	case "ctrl+l":
		m.messages = nil
		m.refreshViewport()
		return nil
	case "pgup":
		m.viewport.PageUp()
		return nil
	case "pgdown":
		m.viewport.PageDown()
		return nil
	}

	if msg.Code == tea.KeyEnter {
		return m.submit()
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

// submit appends the typed prompt, seeds an assistant placeholder, and starts
// the (currently placeholder) agent turn.
func (m *model) submit() tea.Cmd {
	text := strings.TrimSpace(m.input.Value())
	if text == "" || m.busy {
		return nil
	}
	m.input.Reset()

	m.messages = append(m.messages,
		Message{Role: RoleUser, Content: text},
		Message{Role: RoleAssistant},
	)
	m.busy = true
	m.refreshViewport()

	return runPrompt(text)
}

// resize recomputes layout for the current terminal size.
func (m *model) resize() {
	headerH := gloss.Height(m.headerView())
	vpH := m.height - headerH - inputHeight
	if vpH < 1 {
		vpH = 1
	}
	m.viewport.SetWidth(m.width)
	m.viewport.SetHeight(vpH)

	inputW := m.width - inputBoxStyle.GetHorizontalFrameSize()
	if inputW < 1 {
		inputW = 1
	}
	m.input.SetWidth(inputW)

	m.refreshViewport()
}

func (m *model) View() tea.View {
	if !m.ready {
		v := tea.NewView("starting vio…")
		v.AltScreen = true
		return v
	}

	content := strings.Join([]string{
		m.headerView(),
		m.viewport.View(),
		m.inputView(),
	}, "\n")

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// headerView renders the persistent application header: the ASCII wordmark
// beside a metadata column, underlined by a gradient rule.
func (m *model) headerView() string {
	meta := m.metaColumn()

	var block string
	if m.width >= headerArtWidth+headerGap+minMetaWidth {
		block = gloss.JoinHorizontal(gloss.Top, header, strings.Repeat(" ", headerGap), meta)
	} else {
		block = header + "\n" + meta
	}

	return block + "\n" + gradientRule(m.width)
}

// metaColumn renders the header's metadata lines.
func (m *model) metaColumn() string {
	dot, status := idleDot, "idle"
	if m.busy {
		dot, status = busyDot, "thinking"
	}

	rows := []string{
		titleStyle.Render("vio") + " " + metaKeyStyle.Render(version),
		statusStyle.Render("terminal-native coding agent"),
		"",
		metaKeyStyle.Render("model   ") + metaValueStyle.Render(m.modelName),
		metaKeyStyle.Render("cwd     ") + metaValueStyle.Render(m.cwd),
		metaKeyStyle.Render("status  ") + dot + " " + statusStyle.Render(status),
	}
	return strings.Join(rows, "\n")
}

func (m *model) inputView() string {
	style := inputBoxStyle
	if m.input.Focused() {
		style = inputFocusedBoxStyle
	}
	width := m.width
	if width < 1 {
		width = 1
	}
	return style.Width(width).Render(m.input.View())
}
