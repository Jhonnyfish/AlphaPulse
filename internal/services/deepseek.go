package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// DeepSeekClient is an OpenAI-compatible LLM client (works with DeepSeek, local proxies, etc.)
type DeepSeekClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
	log     *zap.Logger
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
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

// NewDeepSeekClient creates a new LLM client. Returns nil if apiKey is empty (LLM disabled).
func NewDeepSeekClient(apiKey, baseURL, model string, log *zap.Logger) *DeepSeekClient {
	if apiKey == "" {
		return nil
	}
	return &DeepSeekClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
		log:     log,
	}
}

// Enabled returns true if the client is configured and ready.
func (d *DeepSeekClient) Enabled() bool {
	return d != nil && d.apiKey != ""
}

// Chat sends a system+user prompt and returns the LLM response text.
func (d *DeepSeekClient) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := chatRequest{
		Model: d.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens: 4096,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := d.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm returned %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 500)]))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if chatResp.Error != nil {
		return "", fmt.Errorf("llm error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("llm returned empty choices")
	}

	content := chatResp.Choices[0].Message.Content
	if d.log != nil {
		d.log.Debug("llm chat completed",
			zap.Int("input_chars", len(systemPrompt)+len(userPrompt)),
			zap.Int("output_chars", len(content)),
		)
	}
	return content, nil
}
