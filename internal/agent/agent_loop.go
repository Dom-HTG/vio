package agent

import (
	"context"
	"fmt"
	"strings"

	"vio/internal/app"
	"vio/internal/domain"
	"vio/internal/model"
	"vio/internal/tools"
)

// loop runs the reason/act/observe cycle until the model stops requesting
// tools or the iteration limit is reached.
func (a *Agent) loop(ctx context.Context, prompt string, emit func(app.Event) bool) {
	if !emit(app.Event{Type: app.EventAgentStarted, Data: app.AgentStarted{Prompt: prompt}}) {
		return
	}

	conversation := a.session.Conversation
	conversation.Messages = append(conversation.Messages, domain.Message{
		Role:    domain.RoleUser,
		Content: prompt,
	})

	for i := 0; i < a.maxIterations; i++ {
		if err := ctx.Err(); err != nil {
			a.emitError(emit, err)
			return
		}
		if !emit(app.Event{Type: app.EventModelStarted, Data: app.ModelStarted{Model: a.model}}) {
			return
		}

		text, calls, err := a.stream(ctx, emit)
		if err != nil {
			a.emitError(emit, err)
			return
		}

		conversation.Messages = append(conversation.Messages, domain.Message{
			Role:      domain.RoleAssistant,
			Content:   text,
			ToolCalls: calls,
		})

		if len(calls) == 0 {
			emit(app.Event{Type: app.EventAgentFinished, Data: app.AgentFinished{Text: text}})
			return
		}

		for _, call := range calls {
			if err := ctx.Err(); err != nil {
				a.emitError(emit, err)
				return
			}
			if !emit(app.Event{Type: app.EventToolStarted, Data: app.ToolStarted{
				ID:   call.ID,
				Name: call.Name,
				Args: call.Arguments,
			}}) {
				return
			}

			output := a.execute(ctx, call)
			content := output.Output
			if content == "" && !output.IsError {
				content = "(no output)"
			}

			if !emit(app.Event{Type: app.EventToolOutput, Data: app.ToolOutput{ID: call.ID, Text: content}}) {
				return
			}
			if !emit(app.Event{Type: app.EventToolFinished, Data: app.ToolFinished{
				ID:   call.ID,
				Name: call.Name,
				Ok:   !output.IsError,
			}}) {
				return
			}

			conversation.Messages = append(conversation.Messages, domain.Message{
				Role:       domain.RoleTool,
				Content:    content,
				ToolCallID: call.ID,
			})
		}
	}

	a.emitError(emit, fmt.Errorf("agent: reached the iteration limit (%d)", a.maxIterations))
}

func (a *Agent) stream(ctx context.Context, emit func(app.Event) bool) (string, []domain.ToolCall, error) {
	if a.provider == nil {
		return "", nil, fmt.Errorf("agent: no model provider configured")
	}

	var definitions []domain.ToolDefinition
	if a.tools != nil {
		definitions = a.tools.Definitions()
	}

	events, err := a.provider.Stream(ctx, a.session.Conversation.Messages, definitions)
	if err != nil {
		return "", nil, err
	}

	var text strings.Builder
	var calls []domain.ToolCall
	for event := range events {
		switch event.Type {
		case model.StreamToken:
			text.WriteString(event.Text)
			if !emit(app.Event{Type: app.EventModelOutput, Data: app.ModelOutput{Text: event.Text}}) {
				return text.String(), calls, ctx.Err()
			}
		case model.StreamToolCall:
			if event.ToolCall != nil {
				calls = append(calls, *event.ToolCall)
			}
		case model.StreamError:
			if event.Error != nil {
				return text.String(), calls, event.Error
			}
			return text.String(), calls, fmt.Errorf("agent: model stream failed")
		case model.StreamDone:
			return text.String(), calls, nil
		}
	}

	if err := ctx.Err(); err != nil {
		return text.String(), calls, err
	}
	return text.String(), calls, nil
}

func (a *Agent) execute(ctx context.Context, call domain.ToolCall) tools.ToolOutput {
	if a.tools == nil {
		return tools.ToolOutput{Output: fmt.Sprintf("%s: no tools available", call.Name), IsError: true}
	}
	tool, ok := a.tools.Get(call.Name)
	if !ok {
		return tools.ToolOutput{Output: fmt.Sprintf("%s: unknown tool", call.Name), IsError: true}
	}

	output, err := tool.Execute(ctx, tools.ToolInput{Args: call.Arguments})
	if err != nil {
		return tools.ToolOutput{Output: fmt.Sprintf("%s: %v", call.Name, err), IsError: true}
	}
	return output
}

func (a *Agent) emitError(emit func(app.Event) bool, err error) {
	emit(app.Event{Type: app.EventError, Data: app.AgentError{Err: err}})
}
