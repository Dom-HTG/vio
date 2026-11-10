package tui

import (
	"context"
	"errors"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	gloss "charm.land/lipgloss/v2"

	"vio/internal/agent"
	"vio/internal/app"
)

type uiState int

const (
	stateIdle uiState = iota
	stateThinking
	stateStreaming
	stateExecuting
	stateWaiting
	stateError
	stateCompleted
	stateCancelled
)

type model struct {
	width  int
	height int
	ready  bool

	input    textinput.Model
	viewport viewport.Model

	messages []Message
	busy     bool
	state    uiState
	runner   agent.AgentRunner

	events          <-chan app.Event
	cancel          context.CancelFunc
	cancelRequested bool
	assistantOpen   bool
	toolResultIndex int

	modelName string
	cwd       string
}

func NewModel(runner agent.AgentRunner) *model {
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
		input:           newInput(),
		viewport:        vp,
		modelName:       modelName,
		cwd:             cwd,
		runner:          runner,
		toolResultIndex: -1,
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

	case eventMsg:
		return m, m.handleEvent(msg.Event)

	case streamDoneMsg:
		if m.busy {
			m.appendStreamClosed()
			m.finishTurn()
		}
		return m, nil
	}

	return m, nil
}

func (m *model) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		if m.busy {
			m.cancelRequested = true
			m.state = stateCancelled
			if m.cancel != nil {
				m.cancel()
			}
			return nil
		}
		return tea.Quit

	case "ctrl+l":
		m.messages = nil
		if !m.busy {
			m.state = stateIdle
		}
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

func (m *model) submit() tea.Cmd {
	text := strings.TrimSpace(m.input.Value())
	if text == "" || m.busy {
		return nil
	}
	if m.runner == nil {
		m.messages = append(m.messages, Message{Role: RoleError, Content: "agent is not configured"})
		m.state = stateError
		m.refreshViewport()
		return nil
	}

	m.input.Reset()
	m.messages = append(m.messages, Message{Role: RoleUser, Content: text})
	m.busy = true
	m.state = stateThinking
	m.cancelRequested = false
	m.assistantOpen = false
	m.toolResultIndex = -1
	m.refreshViewport()

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.events = m.runner.Run(ctx, text)
	return waitForEvent(m.events)
}

func (m *model) handleEvent(event app.Event) tea.Cmd {
	terminal := false

	switch event.Type {
	case app.EventAgentStarted:
		m.state = stateThinking

	case app.EventModelStarted:
		m.state = stateThinking
		m.assistantOpen = false

	case app.EventModelOutput:
		data, ok := event.Data.(app.ModelOutput)
		if ok && data.Text != "" {
			if !m.assistantOpen {
				m.messages = append(m.messages, Message{Role: RoleAssistant})
				m.assistantOpen = true
			}
			m.messages[len(m.messages)-1].Content += data.Text
			m.state = stateStreaming
			m.refreshViewport()
		}

	case app.EventToolStarted:
		data, ok := event.Data.(app.ToolStarted)
		if ok {
			m.messages = append(m.messages, Message{Role: RoleTool, Content: formatToolCall(data)})
			m.toolResultIndex = -1
			m.assistantOpen = false
			m.state = stateExecuting
			m.refreshViewport()
		}

	case app.EventToolOutput:
		data, ok := event.Data.(app.ToolOutput)
		if ok {
			m.messages = append(m.messages, Message{Role: RoleToolResult, Content: truncateOutput(data.Text)})
			m.toolResultIndex = len(m.messages) - 1
			m.refreshViewport()
		}

	case app.EventToolFinished:
		data, ok := event.Data.(app.ToolFinished)
		if ok && !data.Ok && m.toolResultIndex >= 0 && m.toolResultIndex < len(m.messages) {
			m.messages[m.toolResultIndex].Role = RoleError
			m.refreshViewport()
		}
		m.toolResultIndex = -1

	case app.EventAgentFinished:
		data, _ := event.Data.(app.AgentFinished)
		m.finalizeAssistant(data.Text)
		m.state = stateCompleted
		terminal = true

	case app.EventError:
		data, _ := event.Data.(app.AgentError)
		m.appendError(data.Err)
		terminal = true
	}

	if terminal {
		m.refreshViewport()
		m.finishTurn()
		return nil
	}
	if m.events != nil {
		return waitForEvent(m.events)
	}
	return nil
}

func (m *model) finalizeAssistant(text string) {
	if text == "" {
		return
	}
	if n := len(m.messages); n > 0 && m.messages[n-1].Role == RoleAssistant {
		m.messages[n-1].Content = text
		return
	}
	m.messages = append(m.messages, Message{Role: RoleAssistant, Content: text})
}

func (m *model) appendError(err error) {
	if m.cancelRequested || errors.Is(err, context.Canceled) {
		m.messages = append(m.messages, Message{Role: RoleInfo, Content: "cancelled"})
		m.state = stateCancelled
		return
	}
	if err == nil {
		err = errors.New("unknown agent error")
	}
	m.messages = append(m.messages, Message{Role: RoleError, Content: err.Error()})
	m.state = stateError
}

func (m *model) appendStreamClosed() {
	if m.cancelRequested {
		m.messages = append(m.messages, Message{Role: RoleInfo, Content: "cancelled"})
		m.state = stateCancelled
	} else {
		m.messages = append(m.messages, Message{Role: RoleError, Content: "agent stream closed unexpectedly"})
		m.state = stateError
	}
	m.refreshViewport()
}

func (m *model) finishTurn() {
	m.busy = false
	m.events = nil
	m.cancelRequested = false
	m.assistantOpen = false
	m.toolResultIndex = -1
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
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
	switch m.state {
	case stateThinking:
		dot, status = busyDot, "thinking"
	case stateStreaming:
		dot, status = busyDot, "streaming"
	case stateExecuting:
		dot, status = busyDot, "executing tool"
	case stateWaiting:
		dot, status = busyDot, "waiting"
	case stateError:
		dot, status = errorDot, "error"
	case stateCompleted:
		dot, status = idleDot, "completed"
	case stateCancelled:
		dot, status = idleDot, "cancelled"
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
