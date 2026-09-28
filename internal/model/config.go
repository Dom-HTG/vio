package model

import (
	"net/http"
	"os"
)

type ModelConfig struct {
	BaseURL    string
	APIKey     string
	Model      string
	ChatPath   string
	Headers    map[string]string
	Params     map[string]any
	HTTPClient *http.Client
}

func ConfigFromEnv() ModelConfig {
	return ModelConfig{
		BaseURL: os.Getenv("MODEL_BASE_URL"),
		APIKey:  os.Getenv("MODEL_API_KEY"),
		Model:   os.Getenv("MODEL_NAME"),
	}
}
