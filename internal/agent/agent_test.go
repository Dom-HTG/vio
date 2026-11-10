package agent

import (
	"context"
	"errors"
	"testing"

	"vio/internal/app"
	"vio/internal/domain"
	"vio/internal/model"
	"vio/internal/state"
	"vio/internal/tools"
)

type fakeProvider struct {
	streams [][]model.StreamEvent
	seen    [][]domain.Message
	index   int
}

func (f *fakeProvider) Generate(context.Context, []domain.Message, []domain.ToolDefinition) (*model.Response, error) {
	return nil, errors.New("fake: Generate not used")
}

func (f *fakeProvider) Stream(ctx context.Context, msgs []domain.Message, _ []domain.ToolDefinition) (<-chan model.StreamEvent, error) {
	if f.index >= len(f.streams) {
		return nil, errors.New("fake: no scripted stream left")
	}

	f.seen = append(f.seen, append([]domain.Message(nil), msgs...))
	scripted := f.streams[f.index]
	f.index++

	events := make(chan model.StreamEvent)
	go func() {
		defer close(events)
		for _, event := range scripted {
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}()
	return events, nil
}

type fakeTool struct {
	name   string
	output string
	calls  int
	args   []map[string]any
}

func (t *fakeTool) Name() string { return t.name }

func (t *fakeTool) Description() string { return "fake tool" }

func (t *fakeTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Name:        t.name,
		Description: t.Description(),
		Params:      map[string]any{"type": "object"},
	}
}

func (t *fakeTool) Execute(_ context.Context, in tools.ToolInput) (tools.ToolOutput, error) {
	t.calls++
	t.args = append(t.args, in.Args)
	return tools.ToolOutput{Output: t.output}, nil
}

func newTestAgent(t *testing.T, provider model.Provider, registry *tools.Registry, maxIterations int) (*Agent, *state.Conversation) {
	t.Helper()
	conversation := &state.Conversation{}
	agent := NewAgent(AgentConfig{
		Provider:      provider,
		Tools:         registry,
		Session:       &state.Session{Conversation: conversation},
		Model:         "test-model",
		MaxIterations: maxIterations,
	})
	return agent, conversation
}

func collect(events <-chan app.Event) []app.Event {
	var out []app.Event
	for event := range events {
		out = append(out, event)
	}
	return out
}

func eventTypes(events []app.Event) []app.EventType {
	types := make([]app.EventType, len(events))
	for i, event := range events {
		types[i] = event.Type
	}
	return types
}

func TestAgentPlainResponse(t *testing.T) {
	provider := &fakeProvider{streams: [][]model.StreamEvent{
		{
			{Type: model.StreamToken, Text: "hello"},
			{Type: model.StreamDone},
		},
	}}
	agent, conversation := newTestAgent(t, provider, tools.NewRegistry(), 4)

	events := collect(agent.Run(context.Background(), "hi"))
	want := []app.EventType{
		app.EventAgentStarted,
		app.EventModelStarted,
		app.EventModelOutput,
		app.EventAgentFinished,
	}
	if got := eventTypes(events); !equalTypes(got, want) {
		t.Fatalf("event types = %v, want %v", got, want)
	}

	finished, ok := events[len(events)-1].Data.(app.AgentFinished)
	if !ok || finished.Text != "hello" {
		t.Fatalf("finished event = %#v", events[len(events)-1].Data)
	}

	if len(conversation.Messages) != 2 {
		t.Fatalf("conversation has %d messages, want 2", len(conversation.Messages))
	}
	if conversation.Messages[0].Role != domain.RoleUser || conversation.Messages[0].Content != "hi" {
		t.Fatalf("user message = %+v", conversation.Messages[0])
	}
	if conversation.Messages[1].Role != domain.RoleAssistant || conversation.Messages[1].Content != "hello" {
		t.Fatalf("assistant message = %+v", conversation.Messages[1])
	}
}

func TestAgentToolCallThenResponse(t *testing.T) {
	registry := tools.NewRegistry()
	tool := &fakeTool{name: "fake", output: "tool result"}
	if err := registry.Register(tool); err != nil {
		t.Fatal(err)
	}

	provider := &fakeProvider{streams: [][]model.StreamEvent{
		{
			{Type: model.StreamToken, Text: "checking"},
			{Type: model.StreamToolCall, ToolCall: &domain.ToolCall{
				ID:        "call-1",
				Name:      "fake",
				Arguments: map[string]any{"x": "y"},
			}},
			{Type: model.StreamDone},
		},
		{
			{Type: model.StreamToken, Text: "done"},
			{Type: model.StreamDone},
		},
	}}
	agent, conversation := newTestAgent(t, provider, registry, 4)

	events := collect(agent.Run(context.Background(), "go"))
	if tool.calls != 1 {
		t.Fatalf("tool called %d times, want 1", tool.calls)
	}
	if len(tool.args) != 1 || tool.args[0]["x"] != "y" {
		t.Fatalf("tool args = %v", tool.args)
	}

	finished, ok := events[len(events)-1].Data.(app.AgentFinished)
	if !ok || finished.Text != "done" {
		t.Fatalf("finished event = %#v", events[len(events)-1].Data)
	}

	var sawStarted, sawFinished bool
	for _, event := range events {
		switch data := event.Data.(type) {
		case app.ToolStarted:
			sawStarted = data.ID == "call-1" && data.Name == "fake"
		case app.ToolFinished:
			sawFinished = data.ID == "call-1" && data.Ok
		}
	}
	if !sawStarted || !sawFinished {
		t.Fatalf("missing tool events: started=%v finished=%v", sawStarted, sawFinished)
	}

	if len(conversation.Messages) != 4 {
		t.Fatalf("conversation has %d messages, want 4: %+v", len(conversation.Messages), conversation.Messages)
	}
	assistant := conversation.Messages[1]
	if assistant.Role != domain.RoleAssistant || len(assistant.ToolCalls) != 1 {
		t.Fatalf("assistant tool-call message = %+v", assistant)
	}
	result := conversation.Messages[2]
	if result.Role != domain.RoleTool || result.ToolCallID != "call-1" || result.Content != "tool result" {
		t.Fatalf("tool result message = %+v", result)
	}

	if len(provider.seen) != 2 {
		t.Fatalf("provider called %d times, want 2", len(provider.seen))
	}
	if len(provider.seen[1]) != 3 {
		t.Fatalf("second model call saw %d messages, want 3", len(provider.seen[1]))
	}
}

func TestAgentUnknownToolIsFedBackToModel(t *testing.T) {
	provider := &fakeProvider{streams: [][]model.StreamEvent{
		{
			{Type: model.StreamToolCall, ToolCall: &domain.ToolCall{ID: "call-1", Name: "missing"}},
			{Type: model.StreamDone},
		},
		{
			{Type: model.StreamToken, Text: "recovered"},
			{Type: model.StreamDone},
		},
	}}
	agent, conversation := newTestAgent(t, provider, tools.NewRegistry(), 4)

	events := collect(agent.Run(context.Background(), "go"))

	var okFlag bool
	for _, event := range events {
		if data, isFinished := event.Data.(app.ToolFinished); isFinished {
			okFlag = data.Ok
		}
	}
	if okFlag {
		t.Fatal("unknown tool should produce an error result")
	}
	if got := conversation.Messages[2]; got.Role != domain.RoleTool || got.ToolCallID != "call-1" {
		t.Fatalf("tool result = %+v", got)
	}
	finished, ok := events[len(events)-1].Data.(app.AgentFinished)
	if !ok || finished.Text != "recovered" {
		t.Fatalf("finished event = %#v", events[len(events)-1].Data)
	}
}

func TestAgentIterationLimit(t *testing.T) {
	registry := tools.NewRegistry()
	tool := &fakeTool{name: "fake", output: "ok"}
	if err := registry.Register(tool); err != nil {
		t.Fatal(err)
	}

	loop := []model.StreamEvent{
		{Type: model.StreamToolCall, ToolCall: &domain.ToolCall{ID: "call-1", Name: "fake"}},
		{Type: model.StreamDone},
	}
	provider := &fakeProvider{streams: [][]model.StreamEvent{loop, loop}}
	agent, _ := newTestAgent(t, provider, registry, 2)

	events := collect(agent.Run(context.Background(), "go"))
	if tool.calls != 2 {
		t.Fatalf("tool called %d times, want 2", tool.calls)
	}
	if events[len(events)-1].Type != app.EventError {
		t.Fatalf("last event = %v, want %v", events[len(events)-1].Type, app.EventError)
	}
	if _, ok := events[len(events)-1].Data.(app.AgentError); !ok {
		t.Fatalf("error payload = %#v", events[len(events)-1].Data)
	}
}

func equalTypes(a, b []app.EventType) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
