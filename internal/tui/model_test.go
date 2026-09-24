package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func typeString(t *testing.T, m *model, s string) {
	t.Helper()
	for _, r := range s {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func TestSubmitAndStream(t *testing.T) {
	m := NewModel()
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	typeString(t, m, "hello")
	if got := m.input.Value(); got != "hello" {
		t.Fatalf("input = %q, want %q", got, "hello")
	}

	if _, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd == nil {
		t.Fatal("expected a command from submit")
	}

	if len(m.messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(m.messages))
	}
	if m.messages[0].Role != RoleUser || m.messages[0].Content != "hello" {
		t.Fatalf("unexpected user message: %+v", m.messages[0])
	}
	if !m.busy {
		t.Fatal("expected busy after submit")
	}
	if m.input.Value() != "" {
		t.Fatalf("input not reset: %q", m.input.Value())
	}

	m.Update(tokenMsg("world"))
	if got := m.messages[1].Content; got != "world" {
		t.Fatalf("token content = %q, want %q", got, "world")
	}

	m.Update(streamDoneMsg{})
	if m.busy {
		t.Fatal("expected idle after streamDone")
	}

	if v := m.View(); !v.AltScreen || v.Content == "" {
		t.Fatalf("unexpected view: altscreen=%v content=%q", v.AltScreen, v.Content)
	}
}

func TestClearAndEmptySubmit(t *testing.T) {
	m := NewModel()
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
