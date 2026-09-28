package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"vio/internal/domain"
)

const (
	defaultChatPath  = "/chat/completions"
	maxResponseBytes = 8 << 20
	maxStreamLine    = 1 << 20
	maxErrorBody     = 4 << 10
)

// OpenAICompat is a provider for any OpenAI-compatible /chat/completions API.
type OpenAICompat struct {
	baseURL  string
	chatPath string
	apiKey   string
	model    string
	headers  map[string]string
	params   map[string]any
	client   *http.Client
}

func NewOpenAICompat(cfg ModelConfig) (*OpenAICompat, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("%w: base URL is required", ErrConfig)
	}
	path := cfg.ChatPath
	if path == "" {
		path = defaultChatPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	return &OpenAICompat{
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		chatPath: path,
		apiKey:   cfg.APIKey,
		model:    cfg.Model,
		headers:  cfg.Headers,
		params:   cfg.Params,
		client:   client,
	}, nil
}

func (p *OpenAICompat) Generate(ctx context.Context, msgs []domain.Message, tools []domain.ToolDefinition) (*Response, error) {
	req, err := p.newChatRequest(ctx, msgs, tools, false)
	if err != nil {
		return nil, err
	}
	resp, err := p.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body chatResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&body); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if len(body.Choices) == 0 {
		return nil, fmt.Errorf("%w: no choices", ErrInvalidResponse)
	}
	return body.toResponse()
}

func (p *OpenAICompat) Stream(ctx context.Context, msgs []domain.Message, tools []domain.ToolDefinition) (<-chan StreamEvent, error) {
	req, err := p.newChatRequest(ctx, msgs, tools, true)
	if err != nil {
		return nil, err
	}
	resp, err := p.do(req)
	if err != nil {
		return nil, err
	}

	events := make(chan StreamEvent)
	go func() {
		defer close(events)
		defer resp.Body.Close()
		readStream(ctx, resp.Body, events)
	}()
	return events, nil
}

func (p *OpenAICompat) newChatRequest(ctx context.Context, msgs []domain.Message, tools []domain.ToolDefinition, stream bool) (*http.Request, error) {
	wireMsgs := make([]wireMessage, len(msgs))
	for i, m := range msgs {
		wm, err := toWireMessage(m)
		if err != nil {
			return nil, err
		}
		wireMsgs[i] = wm
	}

	payload := map[string]any{
		"model":    p.model,
		"messages": wireMsgs,
		"stream":   stream,
	}
	if len(tools) > 0 {
		payload["tools"] = toWireTools(tools)
	}
	for k, v := range p.params {
		payload[k] = v
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("model: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+p.chatPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHTTP, err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}
	if p.apiKey != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	return req, nil
}

func (p *OpenAICompat) do(req *http.Request) (*http.Response, error) {
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, transportError(req.Context(), err)
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return nil, &HTTPStatusError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}
	return resp, nil
}

func transportError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, context.Canceled) || ctx.Err() == context.Canceled:
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	default:
		return fmt.Errorf("%w: %v", ErrHTTP, err)
	}
}

func readStream(ctx context.Context, body io.Reader, out chan<- StreamEvent) {
	emit := func(ev StreamEvent) bool {
		select {
		case out <- ev:
			return true
		case <-ctx.Done():
			return false
		}
	}

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxStreamLine)

	calls := map[int]*toolCallAccumulator{}
	var order []int

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			emit(StreamEvent{Type: StreamError, Error: fmt.Errorf("%w: %v", ErrInvalidResponse, err)})
			return
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				if !emit(StreamEvent{Type: StreamToken, Text: choice.Delta.Content}) {
					return
				}
			}
			for _, tc := range choice.Delta.ToolCalls {
				acc, ok := calls[tc.Index]
				if !ok {
					acc = &toolCallAccumulator{}
					calls[tc.Index] = acc
					order = append(order, tc.Index)
				}
				acc.id += tc.ID
				acc.name += tc.Function.Name
				acc.args.WriteString(tc.Function.Arguments)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		emit(StreamEvent{Type: StreamError, Error: transportError(ctx, err)})
		return
	}

	for _, idx := range order {
		call, err := calls[idx].result()
		if err != nil {
			emit(StreamEvent{Type: StreamError, Error: err})
			return
		}
		if !emit(StreamEvent{Type: StreamToolCall, ToolCall: &call}) {
			return
		}
	}
	emit(StreamEvent{Type: StreamDone})
}

type toolCallAccumulator struct {
	id   string
	name string
	args strings.Builder
}

func (a *toolCallAccumulator) result() (domain.ToolCall, error) {
	args := map[string]any{}
	if s := a.args.String(); s != "" {
		if err := json.Unmarshal([]byte(s), &args); err != nil {
			return domain.ToolCall{}, fmt.Errorf("%w: %s: %v", ErrInvalidToolCall, a.name, err)
		}
	}
	return domain.ToolCall{ID: a.id, Name: a.name, Args: args}, nil
}

func toWireMessage(m domain.Message) (wireMessage, error) {
	wm := wireMessage{Role: string(m.Role), Content: m.Content, ToolCallID: m.ToolCallID}
	for _, tc := range m.ToolCalls {
		args := tc.Args
		if args == nil {
			args = map[string]any{}
		}
		encoded, err := json.Marshal(args)
		if err != nil {
			return wireMessage{}, fmt.Errorf("%w: %s: %v", ErrInvalidToolCall, tc.Name, err)
		}
		wm.ToolCalls = append(wm.ToolCalls, wireToolCall{
			ID:       tc.ID,
			Type:     "function",
			Function: wireToolFunction{Name: tc.Name, Arguments: string(encoded)},
		})
	}
	return wm, nil
}

func toWireTools(tools []domain.ToolDefinition) []wireTool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]wireTool, len(tools))
	for i, t := range tools {
		params := t.Params
		if params == nil {
			params = map[string]any{}
		}
		out[i] = wireTool{
			Type: "function",
			Function: wireToolSchema{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		}
	}
	return out
}

func toToolCall(c wireToolCall) (domain.ToolCall, error) {
	args := map[string]any{}
	if c.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(c.Function.Arguments), &args); err != nil {
			return domain.ToolCall{}, fmt.Errorf("%w: %s: %v", ErrInvalidToolCall, c.Function.Name, err)
		}
	}
	return domain.ToolCall{ID: c.ID, Name: c.Function.Name, Args: args}, nil
}

type wireMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type wireToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function wireToolFunction `json:"function"`
}

type wireToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type wireTool struct {
	Type     string         `json:"type"`
	Function wireToolSchema `json:"function"`
}

type wireToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   Usage        `json:"usage"`
}

type chatChoice struct {
	Message      wireMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

func (r chatResponse) toResponse() (*Response, error) {
	choice := r.Choices[0]
	out := &Response{
		Message: domain.Message{
			Role:       domain.RoleAssistant,
			Content:    choice.Message.Content,
			ToolCallID: choice.Message.ToolCallID,
		},
		FinishReason:  choice.FinishReason,
		UsageMetadata: r.Usage,
	}
	for _, c := range choice.Message.ToolCalls {
		call, err := toToolCall(c)
		if err != nil {
			return nil, err
		}
		out.ToolCalls = append(out.ToolCalls, call)
		out.Message.ToolCalls = append(out.Message.ToolCalls, call)
	}
	return out, nil
}

type streamChunk struct {
	Choices []streamChoice `json:"choices"`
}

type streamChoice struct {
	Delta streamDelta `json:"delta"`
}

type streamDelta struct {
	Content   string           `json:"content"`
	ToolCalls []streamToolCall `json:"tool_calls"`
}

type streamToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
