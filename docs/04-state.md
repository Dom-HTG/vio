# 04 — State (`internal/state`)

> Sessions, conversation, and history (Issue 10 / Issue 9 session scope).

## Purpose

`internal/state` stores what the agent knows across turns: the current
conversation, the session it belongs to, and enough history to resume or reason
about prior context. It is pure data + in-memory logic; it knows nothing about
models, tools, the UI, or the filesystem beyond its own persistence file. Its
only internal dependency is `internal/domain`, which owns the shared message
types.

Issue 10 proposed a package named `internal/session`; the scaffold named the
package `internal/state` because it hosts session *and* conversation *and*
history. Scaffold naming is canonical.

## Package Layout

```text
internal/state/
├── session.go        Session: one working conversation + metadata
├── conversation.go   ordered messages for one task/turn
└── history.go        prior sessions, resume, on-disk persistence
```

## Models

### Conversation

An ordered list of `domain.Message` values (the provider-independent type from
`internal/domain`, shared with `internal/agent` and `internal/model`). The agent
loop appends user messages, assistant messages, and tool results here as a turn
progresses.

```go
type Conversation struct {
    messages []domain.Message
    limit    int // size guard
}

func (c *Conversation) Add(m domain.Message) error // error when over limit
func (c *Conversation) Messages() []domain.Message
func (c *Conversation) Clear()
```

### Session

One interactive working context (typically one TUI launch = one session). The
session retains the conversation for the process lifetime (Issue 9: "retain the
conversation during the process lifetime") and can be cleared.

```go
type Session struct {
    ID           string
    StartedAt    time.Time
    UpdatedAt    time.Time
    WorkspaceDir string
    Conversation *Conversation
}

func (s *Session) AddMessage(m domain.Message) error
func (s *Session) Messages() []domain.Message
func (s *Session) Clear()
```

### Store

Persistence port for sessions, so the agent and TUI depend on an interface
rather than a concrete on-disk implementation. Implementations contain
conversation data only — never credentials or API keys.

```go
type SessionMeta struct {
    ID           string
    StartedAt    time.Time
    UpdatedAt    time.Time
    WorkspaceDir string
    MessageCount int
}

type Store interface {
    Save(s *Session) error
    Load(id string) (*Session, error)
    List() ([]SessionMeta, error)
    Delete(id string) error
}
```

## Persistence

- Storage is JSON on disk under the user data directory (e.g.
  `%LOCALAPPDATA%\vio\` on Windows, `~/.local/share/vio/` on Unix), keyed by
  session ID, optionally scoped per workspace root.
- State files contain conversation only; never credentials, never API keys.
- v1 keeps persistence minimal and best-effort (Issue 10 does not demand full
  session management polish).

## Context Budget & Truncation

To protect the model context window (Issue 9 large-context protection):

- Enforce a maximum conversation size (message count / total bytes).
- When trimming, prefer dropping the oldest `tool` messages while keeping user
  intent and the latest assistant state coherent.
- Simple byte/line heuristics only — no tokenization in v1.

## Concurrency

A single active turn writes to the conversation. The agent worker goroutine is
the sole writer per turn; reads by the TUI for rendering happen via snapshot
(`Messages()`) to avoid data races. Run `go test -race ./...` to verify (Issue
10).

## Testing (Issue 10)

- Add/read/clear semantics.
- Size-limit enforcement.
- History save/load round-trip to a temp dir.
- No model/tool/UI imports (import-graph check).
