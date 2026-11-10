package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"vio/internal/app"
)

type fakeRunner struct {
	events <-chan app.Event
	ctx    context.Context
}

func (f *fakeRunner) Run(ctx context.Context, _ string) <-chan app.Event {
	f.ctx = ctx
	return f.events
}

func scripted(events ...app.Event) <-chan app.Event {
	ch := make(chan app.Event, len(events))
	for _, event := range events {
		ch <- event
	}
	close(ch)
	return ch
}

func drain(m *model, cmd tea.Cmd) {
	for cmd != nil {
		msg := cmd()
		_, cmd = m.Update(msg)
	}
}

func typeString(t *testing.T, m *model, s string) {
	t.Helper()
	for _, r := range s {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func TestSubmitAndStream(t *testing.T) {
	runner := &fakeRunner{events: scripted(
		app.Event{Type: app.EventAgentStarted, Data: app.AgentStarted{Prompt: "hello"}},
		app.Event{Type: app.EventModelStarted, Data: app.ModelStarted{Model: "test"}},
		app.Event{Type: app.EventModelOutput, Data: app.ModelOutput{Text: "world"}},
		app.Event{Type: app.EventAgentFinished, Data: app.AgentFinished{Text: "world"}},
	)}
	m := NewModel(runner)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	typeString(t, m, "hello")
	if got := m.input.Value(); got != "hello" {
		t.Fatalf("input = %q, want %q", got, "hello")
	}

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from submit")
	}
	if len(m.messages) != 1 || m.messages[0].Role != RoleUser || m.messages[0].Content != "hello" {
		t.Fatalf("unexpected initial messages: %+v", m.messages)
	}
	if !m.busy || m.state != stateThinking {
		t.Fatalf("busy=%v state=%v", m.busy, m.state)
	}
	if m.input.Value() != "" {
		t.Fatalf("input not reset: %q", m.input.Value())
	}

	drain(m, cmd)

	if len(m.messages) != 2 {
		t.Fatalf("got %d messages, want 2: %+v", len(m.messages), m.messages)
	}
	if m.messages[1].Role != RoleAssistant || m.messages[1].Content != "world" {
		t.Fatalf("assistant message = %+v", m.messages[1])
	}
	if m.busy || m.state != stateCompleted {
		t.Fatalf("busy=%v state=%v", m.busy, m.state)
	}

	if v := m.View(); !v.AltScreen || v.Content == "" {
		t.Fatalf("unexpected view: altscreen=%v content=%q", v.AltScreen, v.Content)
	}
}

func TestToolEvents(t *testing.T) {
	runner := &fakeRunner{events: scripted(
		app.Event{Type: app.EventAgentStarted, Data: app.AgentStarted{Prompt: "go"}},
		app.Event{Type: app.EventModelStarted, Data: app.ModelStarted{Model: "test"}},
		app.Event{Type: app.EventToolStarted, Data: app.ToolStarted{
			ID: "call-1", Name: "read_file", Args: map[string]any{"path": "a.go"},
		}},
		app.Event{Type: app.EventToolOutput, Data: app.ToolOutput{ID: "call-1", Text: "package a"}},
		app.Event{Type: app.EventToolFinished, Data: app.ToolFinished{ID: "call-1", Name: "read_file", Ok: true}},
		app.Event{Type: app.EventModelStarted, Data: app.ModelStarted{Model: "test"}},
		app.Event{Type: app.EventModelOutput, Data: app.ModelOutput{Text: "done"}},
		app.Event{Type: app.EventAgentFinished, Data: app.AgentFinished{Text: "done"}},
	)}
	m := NewModel(runner)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	typeString(t, m, "go")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	drain(m, cmd)

	if len(m.messages) != 4 {
		t.Fatalf("got %d messages, want 4: %+v", len(m.messages), m.messages)
	}
	if m.messages[1].Role != RoleTool || !strings.Contains(m.messages[1].Content, "read_file") {
		t.Fatalf("tool call = %+v", m.messages[1])
	}
	if m.messages[2].Role != RoleToolResult || m.messages[2].Content != "package a" {
		t.Fatalf("tool result = %+v", m.messages[2])
	}
	if m.messages[3].Role != RoleAssistant || m.messages[3].Content != "done" {
		t.Fatalf("assistant = %+v", m.messages[3])
	}
	if m.state != stateCompleted || m.busy {
		t.Fatalf("busy=%v state=%v", m.busy, m.state)
	}
}

func TestFailedToolIsRenderedAsError(t *testing.T) {
	runner := &fakeRunner{events: scripted(
		app.Event{Type: app.EventToolStarted, Data: app.ToolStarted{
			ID: "call-1", Name: "shell", Args: map[string]any{"command": "false"},
		}},
		app.Event{Type: app.EventToolOutput, Data: app.ToolOutput{ID: "call-1", Text: "exit code 1"}},
		app.Event{Type: app.EventToolFinished, Data: app.ToolFinished{ID: "call-1", Name: "shell", Ok: false}},
		app.Event{Type: app.EventAgentFinished, Data: app.AgentFinished{Text: ""}},
	)}
	m := NewModel(runner)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	typeString(t, m, "go")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	drain(m, cmd)

	if m.messages[2].Role != RoleError {
		t.Fatalf("failed tool result role = %v, want %v", m.messages[2].Role, RoleError)
	}
}

func TestCancelTurn(t *testing.T) {
	events := make(chan app.Event, 1)
	runner := &fakeRunner{events: events}
	m := NewModel(runner)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	typeString(t, m, "go")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.busy {
		t.Fatal("expected busy")
	}

	m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if runner.ctx.Err() != context.Canceled {
		t.Fatalf("context error = %v, want context.Canceled", runner.ctx.Err())
	}

	events <- app.Event{Type: app.EventError, Data: app.AgentError{Err: context.Canceled}}
	drain(m, cmd)

	if m.busy || m.state != stateCancelled {
		t.Fatalf("busy=%v state=%v", m.busy, m.state)
	}
	last := m.messages[len(m.messages)-1]
	if last.Role != RoleInfo || last.Content != "cancelled" {
		t.Fatalf("last message = %+v", last)
	}
}

func TestClearAndEmptySubmit(t *testing.T) {
	m := NewModel(&fakeRunner{events: scripted()})
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if _, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil {
		t.Fatal("empty submit should not return a command")
	}

	typeString(t, m, "hi")
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(m.messages) == 0 {
		t.Fatal("expected messages before clear")
	}

	m.Update(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl})
	if len(m.messages) != 0 {
		t.Fatalf("clear left %d messages", len(m.messages))
	}
}
