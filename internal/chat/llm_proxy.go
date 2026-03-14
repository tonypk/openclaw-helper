package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMProvider configures an LLM API endpoint.
type LLMProvider struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

// LLMProxy sends chat requests to an LLM API.
type LLMProxy struct {
	providers []LLMProvider
	client    *http.Client
}

// NewLLMProxy creates a proxy with the given providers (tried in order).
func NewLLMProxy(providers []LLMProvider) *LLMProxy {
	return &LLMProxy{
		providers: providers,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Chat sends a message to the LLM and returns the response.
func (p *LLMProxy) Chat(ctx context.Context, systemPrompt string, messages []ChatMessage) (string, error) {
	if len(p.providers) == 0 {
		return "", fmt.Errorf("no LLM providers configured")
	}

	var lastErr error
	for _, provider := range p.providers {
		if provider.APIKey == "" {
			continue
		}
		resp, err := p.callProvider(ctx, provider, systemPrompt, messages)
		if err != nil {
			lastErr = err
			continue
		}
		return resp, nil
	}

	if lastErr != nil {
		return "", fmt.Errorf("all LLM providers failed, last error: %w", lastErr)
	}
	return "", fmt.Errorf("no LLM providers with API keys configured")
}

// HasProviders returns true if at least one provider has an API key.
func (p *LLMProxy) HasProviders() bool {
	for _, prov := range p.providers {
		if prov.APIKey != "" {
			return true
		}
	}
	return false
}

func (p *LLMProxy) callProvider(ctx context.Context, provider LLMProvider, systemPrompt string, messages []ChatMessage) (string, error) {
	// Build OpenAI-compatible request
	apiMsgs := make([]map[string]string, 0, len(messages)+1)
	apiMsgs = append(apiMsgs, map[string]string{
		"role":    "system",
		"content": systemPrompt,
	})
	for _, msg := range messages {
		apiMsgs = append(apiMsgs, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	body := map[string]interface{}{
		"model":    provider.Model,
		"messages": apiMsgs,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	url := provider.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request to %s failed: %w", provider.Name, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s returned %d: %s", provider.Name, resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse %s response: %w", provider.Name, err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("%s returned no choices", provider.Name)
	}

	return result.Choices[0].Message.Content, nil
}
