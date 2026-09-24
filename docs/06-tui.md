# 06 — Terminal UI (`internal/tui`)

> The Bubble Tea application: rendering, input, and the agent-events bridge
> (Issues 1, 8, 9).

## Purpose

`internal/tui` is the user's window into the agent. It renders the conversation,
streams agent activity, collects input, and asks for permission on mutating tool
calls. It never executes model requests, shell commands, or filesystem
operations directly (Issue 8) — it drives `internal/agent` through typed events
and renders the events it receives.

Issue bodies refer to `internal/ui`; the scaffold uses `internal/tui`. Scaffold
naming is canonical.

## Package Layout

```text
internal/tui/
├── model.go       root tea.Model: state machine, view dispatch
├── chat.go        conversation rendering
├── messages.go    tea.Msg types bridging runtime events
├── input.go       text input handling (Enter / Ctrl+C / multiline)
├── styles.go      lipgloss-ish styling
├── tool_view.go   (to be added) tool call/result renderer
└── status.go      (to be added) status indicator per UI state
```

## UI States

The application visually distinguishes (Issue 8):

```text
Idle
Thinking
Streaming
Executing tool
Waiting        (e.g. permission prompt)
Error
Completed
Cancelled
```

The root model holds the current state and renders accordingly; state transitions
come from runtime events (see message mapping below).

## Conversation Rendering (`chat.go`)

Render, visually distinct:

- User messages.
- Assistant messages.
- Tool calls (name + arguments, e.g. `read_file` + path).
- Tool results.
- Errors.

Example rendering (Issue 8):

```text
> Fix the failing tests

● Thinking...

● read_file
  internal/service/service.go

● shell
  go test ./...

✗ TestCreateUser failed

● edit_file
  internal/service/service.go

✓ Tests passing

Done.
```

- Tool output is visually separated from assistant text.
- Large tool output is truncated in the UI while the complete result is kept
  internally (see 01-agent-loop.md event payloads).
- Assistant text streams in incrementally via token events (`EventModelToken`).

## Root Model (`model.go`)

```go
type model struct {
    state     uiState           // one of Idle..Cancelled
    messages  []chatMessage     // view-ready transcript
    input     inputState        // text input model
    style     styles            // shared styling
    runClient agentRunner       // minimal client into internal/agent
    width, height int
}

func (m model) Init() tea.Cmd
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m model) View() string
```

`tea.Cmd`/`tea.Msg` bridging (Issue 7, 8):

- Submitting a prompt returns a `tea.Cmd` that starts the agent turn
  (goroutine) and returns runtime events as messages.
- Runtime events (`Event{...}` from 01-agent-loop.md) arrive as `tea.Msg` and
  are translated in `messages.go` into view updates.

```text
Bubble Tea goroutine  ←  agent worker goroutine (started by tea.Cmd)
        ▲                        │
        └─────── runtime events ─┘
```

## Messages (`messages.go`)

Typed `tea.Msg` wrappers over agent events, e.g.:

```go
type eventMsg struct{ ev agent.Event }
type promptSubmittedMsg struct{ text string }
type toolPermissionMsg struct{ tool string; args map[string]any; approve bool }
type cancelTurnMsg struct{}
```

The `Update` loop consumes these to mutate state, append transcript entries, and
request user decisions.

## Input (`input.go`)

Key bindings (Issue 8):

- `Enter` → submit the current line (or end multiline input).
- `Ctrl+C` → **state-dependent**: quit the app when idle; cancel the running
  turn when an agent is active (maps to context cancellation).
- Clear-input shortcut.
- Multiline input if practical (e.g. `Alt+Enter` for newline).

While a turn is running the UI must stay responsive (Issue 7/8 acceptance:
"UI remains responsive", "agent runs asynchronously").

## Tool & Status Views (`tool_view.go`, `status.go` — to be added)

- `tool_view.go`: renders an active tool call and its streaming output separate
  from the chat transcript; shows truncated output with an indicator.
- `status.go`: compact status line reflecting the current UI state
  (`● Thinking…`, `● shell…`, `Waiting for approval`, `Error`, etc.) and
  permission prompts for mutating/`shell` tools (approve/deny — see 03-tools.md).

## Permission Prompts

On `EventPermissionNeeded`, the UI enters a Waiting state and renders an
approve/deny choice. The user's decision is sent back to the runtime as a
permission event that resumes or aborts the tool call.

## Styles (`styles.go`)

Centralize colors/borders/spacing so the transcript, tool blocks, errors, and
status have a consistent look and can be themed later. Keep Bubble Tea styling
code in this package only.

## Testing (Issue 10)

- Model Update/View tested with synthetic `tea.Msg` sequences covering each UI
  state transition (Idle → Thinking → Streaming → Executing → Completed /
  Error / Cancelled).
- `Ctrl+C` behavior verified per state.
- No import of `internal/model`, `internal/tools`, or `internal/workspace` —
  enforced by the dependency rules in 00-overview.md.
