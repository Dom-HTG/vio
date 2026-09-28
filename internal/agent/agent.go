package agent

import (
	"context"
	"vio/internal/app"
	"vio/internal/model"
	"vio/internal/state"
	"vio/internal/tools"
)

//  the terminal UI communicates directly with the runtime package via agent-events.
// the runtime package is responsible for managing the lifecycle of the agent, including starting and stopping the agent, handling events, and managing the state of the agent.\
// the runtime package depends on the model provider, tools, workspace context & state.
// agent core loop: User prompt -> build context -> send message to model -> model responds -> use tool if need be -> verify output -> repeat until task is complete -> return response to terminal UI.

// the TUI depends on AgentRunner. It is responsible for starting a turn and returns the event stream for that turn.
type AgentRunner interface {
	Run(prompt string, ctx context.Context) <-chan app.Event
}

// agent's dependencies are injected via the config object.
// agentconfig is never constructed internally, only injected.
type AgentConfig struct {
	Provider      model.Provider
	Tools         *tools.Registry
	Session       *state.Session
	MaxIterations int
}

type Agent struct {
	provider      model.Provider
	tools         *tools.Registry
	session       *state.Session
	maxIterations int
}

func NewAgent(config AgentConfig) *Agent {
	return &Agent{
		provider:      config.Provider,
		tools:         config.Tools,
		session:       config.Session,
		maxIterations: config.MaxIterations,
	}
}

func (a *Agent) Run(prompt string, ctx context.Context) <-chan app.Event {
	events := make(chan app.Event)
	go func() {
		defer close(events) // agent creates and closes the channel
		//append prompt to the session conversation
		//build context for the model
		//call provider.Chat with the context and prompt
		//execute any tool calls if needed via the tools registry
		//append the results, repeat until done or max iterations reached
	}()
	return events
}
