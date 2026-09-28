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
├── provider.go   Provider port + StreamEvent + Response
├── openai.go     generic OpenAI-compatible HTTP provider
├── deepseek.go   DeepSeek preset
├── config.go     ModelConfig + env loading
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
    FinishReason  string            // e.g. "stop" | "tool_calls"
    UsageMetadata any               // model.Usage where available
}
```

## Provider Interface

The port is a generate/stream pair over the shared domain types:

```go
type Provider interface {
    Generate(ctx context.Context, msgs []domain.Message, tools []domain.ToolDefinition) (*Response, error)
    Stream(ctx context.Context, msgs []domain.Message, tools []domain.ToolDefinition) (<-chan StreamEvent, error)
}
```

`Generate` is the synchronous, one-shot call. `Stream` returns immediately with
a channel of incremental deltas; the provider creates and closes the channel,
and the agent consumes it.

```go
type StreamEventType string // token | tool_call_delta | done | error

type StreamEvent struct {
    Type     StreamEventType  // which field is meaningful
    Text     string           // for token
    ToolCall *domain.ToolCall // complete call, after accumulating deltas
    Error    error            // in-band terminal error
}
```

`StreamEventType` is provider-local: the agent translates stream events into the
TUI-facing `app.Event` types, so `internal/model` never imports `internal/app`.

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

`OpenAICompat` implements `Provider` for any `/chat/completions` API. It is
transport only; provider-specific defaults live in small presets.

- Use Go's standard `net/http` client (no SDK dependency) — keeps deps minimal
  and works with any OpenAI-compatible server.
- Requests respect `ctx` so cancellation propagates to the HTTP request.
- Parse chat-completions-style responses, including `tool_calls` in assistant
  messages and `tool` role messages carrying results.
- Extensible via `ModelConfig`: `ChatPath` (default `/chat/completions`),
  `Headers` (e.g. Azure's `api-key`), and `Params` (temperature, max_tokens,
  tool_choice, ...). Auth is `Authorization: Bearer <APIKey>` unless the caller
  supplies their own `Authorization` header; an empty `APIKey` is allowed for
  keyless local endpoints.

Presets are thin constructors over the generic transport:

```go
func NewOpenAICompat(cfg ModelConfig) (*OpenAICompat, error) // explicit config
func NewDeepSeek(cfg ModelConfig) (*OpenAICompat, error)     // fills DeepSeek defaults
```

## Configuration (`config.go`)

`ModelConfig` carries the transport settings:

```go
type ModelConfig struct {
    BaseURL    string
    APIKey     string
    Model      string
    ChatPath   string            // default "/chat/completions"
    Headers    map[string]string // extra/override headers
    Params     map[string]any    // extra request-body fields
    HTTPClient *http.Client      // injectable for tests
}
```

`ConfigFromEnv` populates `BaseURL`, `APIKey`, and `Model` from the environment
(Issue 3, 10):

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
- Invalid configuration (`ErrConfig`).
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
