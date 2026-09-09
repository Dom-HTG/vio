# 02 — Model Provider Layer (`internal/model`)

> The BYO-model abstraction (Issues 2, 3, 7). **Note:** this package does not
> exist in the current scaffold and must be created. All other packages already
> reference its types conceptually via the runtime.

## Purpose

`internal/model` lets the agent talk to an LLM *without knowing which provider is
behind it*. v1 ships one working provider: an OpenAI-compatible HTTP provider
that also works against compatible self-hosted/local endpoints (Ollama,
llama.cpp server, LM Studio, OpenRouter, DeepSeek, etc.).

## Package Layout

Issue 3 proposes:

```text
internal/model/
├── model.go      domain types + Provider interface
├── openai.go     OpenAI-compatible HTTP provider
├── config.go     provider configuration from env
└── errors.go     typed provider errors
```

## Domain Types

Provider-independent message types (shared with the agent loop; never coupled to
a vendor SDK):

```go
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
```

Tool definition sent to the model and tool call parsed back out:

```go
type ToolDefinition struct {
    Name        string
    Description string
    Parameters  map[string]any // JSON Schema
}

type ToolCall struct {
    ID        string
    Name      string
    Arguments map[string]any
}

type Response struct {
    Text        string
    ToolCalls   []ToolCall
    FinishReason string   // e.g. "stop" | "tool_calls"
    Usage       Usage      // where available
}

type Usage struct {
    InputTokens  int
    OutputTokens int
}
```

## Provider Interface

Non-streaming entry (Issue 2/3):

```go
type Provider interface {
    Generate(ctx context.Context, msgs []Message, tools []ToolDefinition) (*Response, error)
}
```

Streaming entry (Issue 7): the agent should receive deltas and completion
signals over a channel. Ownership: the provider creates and closes the stream
channel.

```go
type StreamEvent struct {
    Type       StreamEventType // token | tool_call_delta | done | error
    Text       string          // for token
    ToolCall   *ToolCall       // accumulating tool-call deltas
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
2. Translating internal `Message` types to the wire format.
3. Translating tool definitions to the wire format.
4. Parsing model responses.
5. Translating tool calls into internal `ToolCall` types.
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

func (f *FakeProvider) Generate(ctx context.Context, msgs []Message, tools []ToolDefinition) (*Response, error) { ... }
func (f *FakeProvider) Stream(ctx context.Context, msgs []Message, tools []ToolDefinition) (<-chan StreamEvent, error) { ... }
```

## Testing (Issue 10)

- `httptest.Server` round-trip tests against a canned OpenAI-compatible payload
  (text response, tool-call response, error status codes).
- Cancellation: a request made with a cancelled context returns promptly.
- No agent logic lives here — verified by import graph.
