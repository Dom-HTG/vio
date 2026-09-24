# vio Architecture Overview

> Terminal-native, BYO-model coding agent. Inspired by [crush](https://github.com/charmbracelet/crush).

This document is the entry point for the architecture. It describes the overall
system, the responsibilities of each package, and how the rest of the docs in
`docs/` map to the GitHub issue roadmap.

## Purpose & Scope

vio is a coding agent that runs inside a terminal. The user types a task in a
Bubble Tea TUI, the agent reasons about the task using a user-supplied LLM, and
uses a small set of tools to inspect and modify the surrounding repository.

Goals:

- A native terminal experience with an always-responsive UI.
- BYO model: the agent is provider-agnostic and talks to any OpenAI-compatible
  endpoint (including local/self-hosted servers).
- A controlled workspace: all filesystem access is centralized and cannot escape
  the repository root.
- A small, deterministic tool surface the model can drive (`read`, `write`,
  `edit`, `search`, `shell`, `git`).
- Sessions and lightweight repository context so the agent "knows where it is".

Non-goals (v1, from Issue 10):

- Multi-agent systems, MCP, vector databases, RAG.
- Remote execution or distributed workers.
- Complex sandbox infrastructure.
- Autonomous background agents.
- Sophisticated tokenization / context compression.

Those may be considered after the initial v1 release.

## Repository Layout

The project is a single Go module named `vio`.

```text
vio/
├── cmd/agent/               CLI entrypoint
│   ├── main.go              wires everything and starts Bubble Tea
│   └── cli.go               flag parsing and configuration
├── internal/
│   ├── agent/               runtime: core loop, lifecycle, events
│   ├── model/               provider abstraction + HTTP provider   (to be added)
│   ├── state/               session / conversation / history
│   ├── tools/               tool contract, registry, built-in tools
│   ├── tui/                 Bubble Tea application (views + input)
│   └── workspace/           repo-root filesystem abstraction + search
└── pkg/                     reserved for future public/shared packages (empty)
```

`internal/` packages are not importable outside the module. Everything the agent
needs is `internal`; nothing is exported for third parties in v1, which is why
`pkg/` is empty. It is kept as a placeholder and may be dropped if it never gets
used.

## Component Diagram

```text
                    ┌──────────────────────┐
                    │    cmd/agent          │  main.go + cli.go
                    │  flags, config, env   │
                    └──────────┬───────────┘
                               │ starts
                               ▼
                    ┌──────────────────────┐
                    │    internal/tui      │  Bubble Tea (model, chat,
                    │  render + input      │  messages, input, styles)
                    └──────────┬───────────┘
                               │  agent-events (typed messages, no
                               │  direct tool/model/filesystem calls)
                               ▼
                    ┌──────────────────────┐
                    │   internal/agent     │  runtime: owns the loop,
                    │   runtime            │  goroutine, cancellation
                    └──────────┬───────────┘
         ┌─────────────────────┼─────────────────────┐
         ▼                     ▼                     ▼
  ┌───────────────┐   ┌────────────────┐   ┌──────────────────┐
  │ internal/model│   │  internal/tools│   │  internal/state  │
  │ provider only │   │  registry+run  │   │  session/history │
  └───────────────┘   └───────┬────────┘   └──────────────────┘
                              │ rooted in
                              ▼
                    ┌──────────────────┐
                    │ internal/workspace│  path-safe FS + search
                    └──────────────────┘
```

Data flow for one user turn:

1. The user submits a task through the TUI.
2. The TUI forwards it to the agent runtime as an event.
3. The runtime appends the user message to the `state` session and starts a turn.
4. The runtime builds a prompt from system instructions + repository metadata
   (`workspace`) + conversation (`state`) + tool definitions (`tools`/`model`).
5. The runtime calls the `model` provider (streaming).
6. If the model requests tools, the runtime executes them against `workspace` and
   `tools`, appends results to the conversation, and repeats.
7. When the model finishes, the runtime emits completion events back to the TUI.

## Package Responsibilities & Dependency Rules

| Package | Responsibility | May depend on |
| --- | --- | --- |
| `cmd/agent` | Parse flags/env, build config, construct dependencies, start TUI | all `internal/*` |
| `internal/tui` | Render conversation, capture input, display agent state; never executes tools/models/FS directly | `internal/agent` (events only) |
| `internal/agent` | Agent lifecycle, core loop, event emission, cancellation, context building | `internal/model`, `internal/tools`, `internal/state`, `internal/workspace` |
| `internal/model` | Provider abstraction, message translation, HTTP transport | stdlib only |
| `internal/tools` | Tool contract + registry + built-ins | `internal/workspace` |
| `internal/state` | Session, conversation, history in-memory + persistence | stdlib only |
| `internal/workspace` | Path-safe repo FS, listing, search | stdlib only |

Hard rules:

- `internal/tui` must never import `internal/model`, `internal/tools`,
  `internal/workspace`, Bubble Tea SDKs' agent logic, or any provider SDK. It
  communicates with the runtime exclusively through typed agent-events.
- `internal/model`, `internal/tools`, `internal/state`, and
  `internal/workspace` must never import Bubble Tea or any provider SDK.
- `internal/agent` must have no Bubble Tea dependency.
- The domain types (messages, roles, tool calls, events) live in `internal/agent`
  and `internal/model` and must not be coupled to a specific vendor SDK.
- All filesystem access performed by tools goes through `internal/workspace`.
- All shell execution happens through the `shell` tool, rooted at the workspace.

## GitHub Issue Correlation

The roadmap is tracked as GitHub issues. Each issue's scope maps to the packages
and documents below. Naming note: some issue bodies propose package paths that
differ from the scaffold; the scaffold layout above is canonical and the docs
follow it.

| Issue | Title | Primary packages | Doc |
| --- | --- | --- | --- |
| #1 | Bootstrap CLI + Bubble Tea shell | `cmd/agent`, `internal/tui` | 00-overview, 06-tui |
| #2 | Core domain & agent interfaces | `internal/agent`, `internal/model`, `internal/tools` | 01, 02, 03 |
| #3 | BYO model provider layer | `internal/model` | 02 |
| #4 | Workspace + filesystem abstraction | `internal/workspace` | 05 |
| #5 | Initial coding tools + registry | `internal/tools` | 03 |
| #6 | Agent reasoning / tool loop | `internal/agent` | 01 |
| #7 | Streaming, events, goroutines, cancellation | `internal/agent`, `internal/model` | 01, 02 |
| #8 | Streaming… (duplicate of #7, closed) | — | 01, 02 |
| #9 | Terminal-native coding experience | `internal/tui` | 06 |
| #10 | Sessions, Git awareness, repo context | `internal/state`, `internal/tools`, `internal/workspace` | 04, 03, 05 |
| #11 | Harden, test, package, v1 release | repo-wide (tests, config, logging, docs) | 00-overview + all |

### Scaffold ↔ issue naming drift

The scaffold was bootstrapped before (or independently of) the issues and uses
names that differ from some issue proposals. Decisions:

- `internal/tui` instead of issue-proposed `internal/ui` — kept, more precise.
- `internal/state` instead of issue-proposed `internal/session` — kept;
  `state` also hosts `conversation` and `history`, not just sessions.
- Git awareness lives as `git_status` / `git_diff` **tools** in
  `internal/tools` rather than a separate `internal/git` package.
- Repository/context concerns live in `internal/workspace` rather than a
  separate `internal/context`; the prompt context builder is part of
  `internal/agent` (see 01-agent-loop.md).
- Provider layer (`internal/model`) and an agent worker/runtime split
  (`internal/runtime`) referenced in issues are folded into `internal/model`
  (new, see 02-model-provider.md) and `internal/agent` (existing `runtime.go`).
- `internal/tui` will grow `tool_view.go` and `status.go` (see 06-tui.md).
- `internal/workspace` will grow `errors.go`.

These are intentional; do not rename existing packages to match issue bodies
without an explicit decision.

## Configuration & Environment

Configuration comes from environment variables (per Issue 10). A config file is
a post-v1 nicety; do not hard-code credentials.

| Variable | Purpose |
| --- | --- |
| `MODEL_BASE_URL` | Base URL of the (OpenAI-compatible) API endpoint |
| `MODEL_API_KEY` | API key / bearer token for the endpoint |
| `MODEL_NAME` | Model identifier sent in requests |
| `AGENT_MAX_ITERATIONS` | Hard cap on model round-trips per turn |
| `AGENT_TOOL_TIMEOUT` | Default per-tool execution timeout |

Flags on `cmd/agent` (see `cli.go`) supplement these and may override defaults,
e.g. `--debug` for runtime diagnostics and `--cwd`/positional for the workspace
root.

## Logging

Normal terminal output is the TUI and must stay clean. Optional debug logging
(`agent --debug`) writes structured runtime diagnostics (agent events, tool
lifecycle, HTTP statuses) to a log file or stderr, never interleaved with the
rendered UI.

## Testing Strategy

Per Issue 10, correctness is measured with:

- Unit tests for `workspace`, `tools`, `tool registry`, `model provider`,
  `agent loop`, `state`, git tools, and event handling.
- A fake/mock provider (`FakeProvider`) so agent behavior is testable without
  real network calls:

```go
type FakeProvider struct {
    Responses []model.Response
}
```

- `go test ./...` and `go test -race ./...` must pass; no obvious goroutine
  leaks or channel-lifecycle bugs.
- The application must not panic on normal operational failures (network,
  timeout, invalid response, tool/shell/file errors, cancellation, iteration
  limit).

## Document Index

| Doc | Covers |
| --- | --- |
| 01-agent-loop.md | `internal/agent` runtime: loop, events, concurrency, cancellation |
| 02-model-provider.md | `internal/model`: provider contract, OpenAI-compatible transport |
| 03-tools.md | `internal/tools`: contract, registry, built-ins |
| 04-state.md | `internal/state`: session, conversation, history, persistence |
| 05-workspace.md | `internal/workspace`: filesystem safety, search, repo context |
| 06-tui.md | `internal/tui`: views, input, event plumbing |
