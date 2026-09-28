package model

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vio/internal/domain"
)

// Live smoke test against the real API. Skipped unless MODEL_API_KEY is set.
// Run: go test ./internal/model -run TestLive -v
// Reads .env from the repo root; real environment variables take precedence.

func loadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for i := 0; i < 5; i++ {
		data, err := os.ReadFile(filepath.Join(dir, ".env"))
		if err == nil {
			applyEnvFile(string(data))
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func applyEnvFile(content string) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func liveProvider(t *testing.T) *OpenAICompat {
	t.Helper()
	loadDotEnv()
	if os.Getenv("MODEL_API_KEY") == "" {
		t.Skip("set MODEL_API_KEY in .env to run live tests")
	}
	p, err := NewDeepSeek(ConfigFromEnv())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func weatherTool() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name:        "get_weather",
		Description: "Get the current weather for a city.",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]any{"type": "string", "description": "City name"},
			},
			"required": []any{"city"},
		},
	}
}

func TestLiveGenerateText(t *testing.T) {
	p := liveProvider(t)
	resp, err := p.Generate(context.Background(), []domain.Message{
		{Role: domain.RoleUser, Content: "Reply with exactly one word: pong"},
	}, nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	t.Logf("text=%q finish=%q", resp.Text, resp.FinishReason)
	if strings.TrimSpace(resp.Text) == "" {
		t.Fatal("empty text")
	}
}

func TestLiveGenerateToolCall(t *testing.T) {
	p := liveProvider(t)
	resp, err := p.Generate(context.Background(), []domain.Message{
		{Role: domain.RoleUser, Content: "What is the weather in Paris? Call the tool."},
	}, []domain.ToolDefinition{weatherTool()})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	t.Logf("toolcalls=%d finish=%q", len(resp.ToolCalls), resp.FinishReason)
	if len(resp.ToolCalls) == 0 {
		t.Fatal("expected a tool call")
	}
	c := resp.ToolCalls[0]
	t.Logf("call id=%q name=%q args=%v", c.ID, c.Name, c.Arguments)
	if c.Name != "get_weather" || c.Arguments["city"] == "" {
		t.Fatalf("unexpected call: %+v", c)
	}
}

func TestLiveStream(t *testing.T) {
	p := liveProvider(t)
	ch, err := p.Stream(context.Background(), []domain.Message{
		{Role: domain.RoleUser, Content: "Count from 1 to 5, separated by spaces."},
	}, nil)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	var text strings.Builder
	var done bool
	for ev := range ch {
		switch ev.Type {
		case StreamToken:
			text.WriteString(ev.Text)
		case StreamDone:
			done = true
		case StreamError:
			t.Fatalf("stream error: %v", ev.Error)
		}
	}
	t.Logf("streamed=%q done=%v", text.String(), done)
	if text.Len() == 0 || !done {
		t.Fatal("stream incomplete")
	}
}
