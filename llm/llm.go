package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Config holds LLM provider configuration, read from environment variables.
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

// ConfigFromEnv reads LLM configuration from environment variables.
func ConfigFromEnv() (*Config, error) {
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY not set")
	}
	base := os.Getenv("DEEPSEEK_BASE_URL")
	if base == "" {
		base = "https://api.deepseek.com"
	}
	model := os.Getenv("DEEPSEEK_MODEL")
	if model == "" {
		model = "deepseek-v4-pro"
	}
	return &Config{APIKey: key, BaseURL: base, Model: model}, nil
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ChatCompletion sends messages to the LLM and returns the response content.
func ChatCompletion(cfg *Config, messages []Message) (string, error) {
	body := chatRequest{
		Model:    cfg.Model,
		Messages: messages,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := cfg.BaseURL + "/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return result.Choices[0].Message.Content, nil
}
