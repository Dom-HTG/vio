package agent
//  the terminal UI communicates directly with the runtime package via agent-events.
// the runtime package is responsible for managing the lifecycle of the agent, including starting and stopping the agent, handling events, and managing the state of the agent.\
// the runtime package depends on the model provider, tools, workspace context & state.
// agent core loop: User prompt -> build context -> send message to model -> model responds -> use tool if need be -> verify output -> repeat until task is complete -> return response to terminal UI.

type Agent struct {
	// agent dependencies
}