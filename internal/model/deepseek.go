package model

import "fmt"

const (
	deepseekBaseURL = "https://api.deepseek.com"
	deepseekModel   = "deepseek-chat"
)

func NewDeepSeek(cfg ModelConfig) (*OpenAICompat, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("%w: MODEL_API_KEY is required", ErrAuth)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = deepseekBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = deepseekModel
	}
	return NewOpenAICompat(cfg)
}
