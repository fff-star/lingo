package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	url := cfg.BaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	requestSize := len(data)
	// 20-minute timeout: large articles with reasoning models can take a long time.
	timeout := 1200 * time.Second
	fmt.Fprintf(os.Stderr, "→ Connected to %s (model: %s), request size: %d bytes, waiting for response (timeout: %v)...\n", cfg.BaseURL, cfg.Model, requestSize, timeout)

	start := time.Now()
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request (after %v): %w", time.Since(start).Round(time.Second), err)
	}
	defer resp.Body.Close()

	fmt.Fprintf(os.Stderr, "→ Response headers received after %v (status: %d), reading body...\n", time.Since(start).Round(time.Second), resp.StatusCode)

	pr := &progressReader{r: resp.Body, start: start}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, pr); err != nil {
		return "", fmt.Errorf("read response (after %v): %w", time.Since(start).Round(time.Second), err)
	}
	elapsed := time.Since(start).Round(time.Second)
	responseSize := buf.Len()
	fmt.Fprintf(os.Stderr, "→ Response received (%d bytes, %d chunks) after %v, parsing...\n", responseSize, pr.chunks, elapsed)

	var result chatResponse
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		preview := buf.String()
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		return "", fmt.Errorf("parse response: %w\n\nRaw response preview:\n%s", err, preview)
	}
	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return result.Choices[0].Message.Content, nil
}

// progressReader wraps a reader and periodically reports read progress to stderr.
type progressReader struct {
	r       io.Reader
	start   time.Time
	read    int64
	chunks  int
	lastLog time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.read += int64(n)
	pr.chunks++
	if n > 0 && time.Since(pr.lastLog) > 3*time.Second {
		fmt.Fprintf(os.Stderr, "→ Reading... %d bytes in %d chunks after %v\n",
			pr.read, pr.chunks, time.Since(pr.start).Round(time.Second))
		pr.lastLog = time.Now()
	}
	return n, err
}
