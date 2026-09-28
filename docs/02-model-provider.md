# 02 — Model Provider Layer (`internal/model`)

> The BYO-model abstraction (Issues 2, 3, 7). The provider-independent message
> and tool types now live in `internal/domain`; this package defines the provider
> port and the OpenAI-compatible transport that implements it.

## Purpose

`internal/model` lets the agent talk to an LLM *without knowing which provider is
behind it*. v1 ships one working provider: an OpenAI-compatible HTTP provider
that also works against compatible self-hosted/local endpoints (Ollama,
llama.cpp server, LM Studio, OpenRouter, DeepSeek, etc.).

## Package Layout

Issue 3 proposes:

```text
internal/model/
├── provider.go   Provider port + Request/Response
├── openai.go     OpenAI-compatible HTTP provider
├── config.go     provider configuration from env
└── errors.go     typed provider errors
```

## Domain Types

The provider-independent message and tool types live in `internal/domain` and are
shared by `agent`, `model`, `state`, and `tools`. They are never coupled to a
vendor SDK:

```go
// internal/domain
type Role string

const (
    RoleSystem    Role = "system"
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
    RoleTool      Role = "tool"
)

type Message struct {
    Role       Role
    Content    string
    ToolCalls  []ToolCall
    ToolCallID string
}

type ToolCall struct {
    ID   string
    Name string
    Args map[string]any
}

type ToolDefinition struct {
    Name        string
    Description string
    Params      map[string]any // JSON Schema
}
```

`internal/model` wraps those types at its boundary:

```go
// internal/model
type Request struct {
    Model    string
    Messages []domain.Message
}

type Response struct {
    Message       domain.Message
    ToolCalls     []domain.ToolCall // tool calls the model requested, if any
    UsageMetadata any
}
```

## Provider Interface

Non-streaming entry (Issue 2/3). The scaffold's concrete port is a single
request/response call:

```go
type Provider interface {
    Chat(ctx context.Context, request Request) (Response, error)
}
```

The roadmap's more granular shape passes messages and tool definitions
explicitly:

```go
type Provider interface {
    Generate(ctx context.Context, msgs []domain.Message, tools []domain.ToolDefinition) (*Response, error)
}
```

Streaming entry (Issue 7): the agent should receive deltas and completion
signals over a channel. Ownership: the provider creates and closes the stream
channel.

```go
type StreamEvent struct {
    Type       StreamEventType  // token | tool_call_delta | done | error
    Text       string           // for token
    ToolCall   *domain.ToolCall // accumulating tool-call deltas
    Error      error
}

type Provider interface {
    Generate(ctx, msgs, tools) (*Response, error)
    Stream(ctx, msgs, tools) (<-chan StreamEvent, error)
}
```

Streaming must support text deltas, tool-call deltas where the provider supports
them, completion, and errors.

## Provider Responsibilities

The provider is responsible for:

1. Constructing API requests.
2. Translating `domain.Message` types to the wire format.
3. Translating `domain.ToolDefinition` values to the wire format.
4. Parsing model responses.
5. Translating tool calls into `domain.ToolCall` types.
6. Handling HTTP errors and mapping them to typed errors.
7. Respecting `context.Context` for cancellation and timeouts.

It must **not**:

- Execute tools.
- Modify files.
- Know about Bubble Tea.
- Run the agent loop.

## OpenAI-Compatible HTTP Provider (`openai.go`)

- Use Go's standard `net/http` client (no SDK dependency) — keeps deps minimal
  and works with any OpenAI-compatible server.
- Requests respect `ctx` so cancellation propagates to the HTTP request.
- Parse chat-completions-style responses, including `tool_calls` in assistant
  messages and `tool` role messages carrying results.

## Configuration (`config.go`)

Provider configuration comes from environment variables (Issue 3, 10):

```text
MODEL_BASE_URL    endpoint base URL
MODEL_API_KEY     API key / bearer token
MODEL_NAME        model identifier
```

Never hard-code credentials. API keys come from the environment only.

## Error Taxonomy (`errors.go`)

Create meaningful, typed errors for (Issue 3, 10):

- Authentication failures.
- HTTP / network failures.
- Timeouts.
- Invalid / malformed responses.
- Malformed tool calls in the response.
- Context cancellation (`context.Canceled` passthrough).

Errors are exported so `internal/agent` can decide whether a failure is
recoverable, terminal, or retryable.

## Fake Provider for Tests (Issue 10)

Agent tests must not make network calls. Ship a test double in the package (or a
test-only file):

```go
type FakeProvider struct {
    Responses []Response          // scripted responses, consumed in order
    Streams   []<-chan StreamEvent // optional scripted streams
    Err       error
}

func (f *FakeProvider) Generate(ctx context.Context, msgs []domain.Message, tools []domain.ToolDefinition) (*Response, error) { ... }
func (f *FakeProvider) Stream(ctx context.Context, msgs []domain.Message, tools []domain.ToolDefinition) (<-chan StreamEvent, error) { ... }
```

## Testing (Issue 10)

- `httptest.Server` round-trip tests against a canned OpenAI-compatible payload
  (text response, tool-call response, error status codes).
- Cancellation: a request made with a cancelled context returns promptly.
- No agent logic lives here — verified by import graph.
