package agent

import (
	"context"

	"vio/internal/app"
	"vio/internal/model"
	"vio/internal/state"
	"vio/internal/tools"
)

const defaultMaxIterations = 16

// AgentRunner is the contract the TUI depends on.
type AgentRunner interface {
	Run(ctx context.Context, prompt string) <-chan app.Event
}

type AgentConfig struct {
	Provider      model.Provider
	Tools         *tools.Registry
	Session       *state.Session
	Model         string
	MaxIterations int
}

type Agent struct {
	provider      model.Provider
	tools         *tools.Registry
	session       *state.Session
	model         string
	maxIterations int
}

func NewAgent(config AgentConfig) *Agent {
	if config.MaxIterations <= 0 {
		config.MaxIterations = defaultMaxIterations
	}
	if config.Session == nil {
		config.Session = &state.Session{}
	}
	if config.Session.Conversation == nil {
		config.Session.Conversation = &state.Conversation{}
	}

	return &Agent{
		provider:      config.Provider,
		tools:         config.Tools,
		session:       config.Session,
		model:         config.Model,
		maxIterations: config.MaxIterations,
	}
}

// Run starts one turn and returns its event stream. The agent owns the channel
// and closes it when the turn ends.
func (a *Agent) Run(ctx context.Context, prompt string) <-chan app.Event {
	events := make(chan app.Event)
	go func() {
		defer close(events)
		a.loop(ctx, prompt, func(event app.Event) bool {
			select {
			case events <- event:
				return true
			case <-ctx.Done():
				return false
			}
		})
	}()
	return events
}
