package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"vio/internal/agent"
	"vio/internal/model"
	"vio/internal/state"
	"vio/internal/tools"
	"vio/internal/workspace"
)

type stack struct {
	Workspace *workspace.Local
	Provider  model.Provider
	Tools     *tools.Registry
	Session   *state.Session
	Agent     *agent.Agent
}

func newStack() (*stack, error) {
	ws, err := workspace.NewFromCWD()
	if err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}

	loadDotEnv(ws.Root())

	cfg := model.ConfigFromEnv()
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("MODEL_API_KEY is not set; add it to .env or export it")
	}

	provider, err := newProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("provider: %w", err)
	}

	registry, err := newRegistry(ws)
	if err != nil {
		return nil, fmt.Errorf("tools: %w", err)
	}

	maxIterations, err := maxIterationsFromEnv()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session := &state.Session{
		ID:           fmt.Sprintf("session-%d", now.UnixNano()),
		StartedAt:    now,
		UpdatedAt:    now,
		WorkspaceDir: ws.Root(),
		Conversation: &state.Conversation{},
	}

	runtime := agent.NewAgent(agent.AgentConfig{
		Provider:      provider,
		Tools:         registry,
		Session:       session,
		Model:         cfg.Model,
		MaxIterations: maxIterations,
	})

	return &stack{
		Workspace: ws,
		Provider:  provider,
		Tools:     registry,
		Session:   session,
		Agent:     runtime,
	}, nil
}

func newProvider(cfg model.ModelConfig) (model.Provider, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return model.NewDeepSeek(cfg)
	}
	return model.NewOpenAICompat(cfg)
}

func newRegistry(ws workspace.FS) (*tools.Registry, error) {
	registry := tools.NewRegistry()
	for _, tool := range []tools.Tool{
		tools.NewReadFile(ws),
		tools.NewWriteFile(ws),
		tools.NewEditFile(ws),
		tools.NewSearch(ws),
		tools.NewShell(ws),
	} {
		if err := registry.Register(tool); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func maxIterationsFromEnv() (int, error) {
	value := strings.TrimSpace(os.Getenv("AGENT_MAX_ITERATIONS"))
	if value == "" {
		return 0, nil
	}
	iterations, err := strconv.Atoi(value)
	if err != nil || iterations <= 0 {
		return 0, fmt.Errorf("AGENT_MAX_ITERATIONS must be a positive integer, got %q", value)
	}
	return iterations, nil
}
