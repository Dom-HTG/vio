# 01 — Agent Runtime (`internal/agent`)

> The heart of vio. Owns the agent lifecycle, the reasoning/tool loop, the event
> stream that feeds the TUI, and per-turn cancellation.

## Responsibilities

`internal/agent` is the single place that decides *what the agent does next*.
It does not render anything, does not talk to HTTP endpoints, and does not touch
the filesystem directly. It orchestrates:

- `internal/model` — to talk to the LLM.
- `internal/tools` — to execute tool calls the model requests.
- `internal/state` — to store and read the conversation.
- `internal/workspace` — to gather repository context for prompts.

The current scaffold lives in `runtime.go` (package `agent`) with an `Agent`
struct placeholder. Its documented contract:

> The terminal UI communicates directly with the runtime package via
> agent-events. The runtime is responsible for managing the lifecycle of the
> agent, including starting and stopping the agent, handling events, and
> managing the state of the agent.

Issue texts (#2, #6, #7) suggest splitting this across `agent.go`, `loop.go`,
`context.go`, `errors.go`, and `events.go`. Prefer those file names inside
`internal/agent` as the package grows:

```text
internal/agent/
├── runtime.go      Agent struct, lifecycle, event channel ownership
├── loop.go         the core turn loop (Issue 6)
├── context.go      prompt/context builder (Issue 6, 10)
├── events.go       event types emitted to the UI (Issue 7)
└── errors.go       typed error taxonomy (Issue 6)
```

## Core Loop

The loop formalizes the comment in `runtime.go`:

> User prompt → build context → send message to model → model responds → use tool
> if need be → verify output → repeat until task is complete → return response to
> terminal UI.

```text
User prompt
    ↓
Add user message to session
    ↓
Build context (system + repo metadata + conversation + tool definitions)
    ↓
Send messages + tool definitions to model
    ↓
Receive response
    ↓
Tool calls?
 ┌──┴───┐
 No     Yes
 │       │
 ▼       ▼
Done   For each tool call:
       1. validate tool exists
       2. validate arguments against schema
       3. execute tool (rooted in workspace)
       4. capture result (success or structured error)
       5. add tool result to conversation
       6. emit ToolResult event
        │
        ▼
        Call model again
```

## Agent Types

Core types (provider-independent; shared conceptually with `internal/model`):

```go
type Role string

const (
    RoleSystem    Role = "system"
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
    RoleTool      Role = "tool"
)

type ToolCall struct {
    ID        string
    Name      string
    Arguments map[string]any
}

type Message struct {
    Role       Role
    Content    string
    ToolCalls  []ToolCall
    ToolCallID string
}
```

The agent entrypoint used by the TUI (Issue 2's `Runner`):

```go
type Runner interface {
    Run(context.Context, string) error
}
```

`Agent` wires the loop's dependencies together:

```go
type Agent struct {
    provider model.Provider
    tools    *tools.Registry
    session  *state.Session
    events   chan<- Event
    maxIterations int
}
```

## Loop Protection

The loop must never run forever (Issue 6):

- `MaxIterations` — hard cap on model round-trips per turn
  (`AGENT_MAX_ITERATIONS`).
- `MaxToolCalls` — optional cap on tool calls per turn.

Crossing a limit terminates the turn with an iteration-limit error surfaced as an
event rather than silently stopping.

## Context Builder (`context.go`)

Every model request receives (Issue 6, 10):

```text
system instructions
+ repository metadata (workspace)   -- git branch, status, tree overview
+ conversation (state)              -- trimmed/limited to a budget
+ tool definitions (tools → model)
```

The builder must stay deterministic and lightweight. It is responsible for the
"large context protection" limits from Issue 9:

- Maximum file output.
- Maximum tool-result size.
- Maximum repository tree size.
- Maximum conversation size (number of messages / total bytes).

Do not implement sophisticated tokenization yet; byte/line heuristics are fine.

## Events (`events.go`)

The runtime emits events through a channel it owns; the TUI consumes them
(Issue 7, 9). Event types:

```go
type EventType string

const (
    EventAgentStarted    EventType = "agent_started"
    EventModelStarted    EventType = "model_started"
    EventModelToken      EventType = "model_token"      // streaming delta
    EventModelFinished   EventType = "model_finished"
    EventToolStarted     EventType = "tool_started"
    EventToolOutput      EventType = "tool_output"      // streaming tool output
    EventToolFinished    EventType = "tool_finished"
    EventAgentFinished   EventType = "agent_finished"
    EventPermissionNeeded EventType = "permission_needed"
    EventError           EventType = "error"
    EventCancelled       EventType = "cancelled"
)

type Event struct {
    Type EventType
    Data any
}
```

Event payloads carry only UI-relevant data (e.g. `EventModelToken{Text}`,
`EventToolStarted{Name, Args}`, `EventToolFinished{Name, Ok, Truncated}`).
Full tool output stays in `internal/state`/results; the UI receives a
truncated view but the complete result is preserved internally (Issue 8).

## Concurrency & Channel Ownership (Issue 7)

```text
Bubble Tea goroutine
        │
        │ starts turn
        ▼
Agent worker goroutine
        │
        ▼
event channel (runtime-owned)
        │
        ▼
Bubble Tea consumes via tea.Cmd
```

Ownership rules (documented explicitly because Issue 7 calls them out):

- The **agent runtime** creates the event channel per turn (or owns a persistent
  one) and is the *only* goroutine that closes it.
- The **TUI** only receives; it never closes the channel.
- Exactly one producer per turn: the agent worker goroutine. Tool executions may
  run on the worker or a small bounded pool, but all writes funnel through the
  worker to avoid concurrent sends.
- No goroutine leaks: every turn goroutine must exit when its context is
  cancelled or the loop completes.

## Cancellation

Each turn runs under a cancellable context derived from the agent's lifetime
context (Issue 7):

```text
Ctrl+C / cancel
      ↓
context.Cancel()
      ↓
model request cancelled   (HTTP request respects ctx)
      ↓
shell command cancelled   (exec.CommandContext)
      ↓
agent loop stops → EventCancelled
```

Cancellation must propagate to: model HTTP requests (`internal/model`), shell
commands (`shell` tool via `exec.CommandContext`), and any in-flight tool. The
TUI's `Ctrl+C` semantics depend on state (see 06-tui.md): quit when idle, cancel
the running turn when busy.

## Error Taxonomy (`errors.go`, Issue 6)

Distinguish and handle distinctly:

```text
model error
tool error
invalid tool call
iteration limit
context cancellation
```

A **tool failure is not fatal**: it is returned to the model as a structured
tool result so the model can recover (e.g. `edit_file` failed to find
`old_text`). Model/transport errors and iteration limits terminate the turn.

## Testing (Issue 10)

`internal/agent` is tested with a fake provider (see 02-model-provider.md) that
replays scripted responses covering: single response, tool call → result →
follow-up, multi-step tasks, tool-error recovery, iteration-limit stop, and
cancellation. No real network in tests.
