package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const APIStyleOpenAIResponses = "openai-responses"

// ResponseRequest is the minimum provider execution request for OpenAI Responses.
type ResponseRequest struct {
	Provider        string
	Model           string
	Input           string
	MaxOutputTokens int
}

// Usage captures the minimum token accounting returned by the provider.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ResponseResult is the normalized provider execution result.
type ResponseResult struct {
	ResponseID string `json:"response_id"`
	Model      string `json:"model"`
	OutputText string `json:"output_text"`
	Usage      Usage  `json:"usage"`
	APIStyle   string `json:"api_style"`
}

// Client executes provider-backed model calls.
type Client struct {
	registry *Registry
	http     http.Client
}

// NewClient constructs a provider client backed by a registry.
func NewClient(registry *Registry) *Client {
	if registry == nil {
		registry = NewRegistry("")
	}
	return &Client{
		registry: registry,
		http: http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ResolveProvider selects an enabled provider config by explicit kind or default fallback.
func (c *Client) ResolveProvider(kind string) (Config, error) {
	return c.registry.ResolveForExecution(kind)
}

// CreateResponse executes a minimum non-streaming OpenAI Responses request.
func (c *Client) CreateResponse(ctx context.Context, request ResponseRequest) (ResponseResult, error) {
	cfg, err := c.ResolveProvider(request.Provider)
	if err != nil {
		return ResponseResult{}, err
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return ResponseResult{}, fmt.Errorf("provider base URL is not configured")
	}
	if strings.TrimSpace(cfg.AuthToken) == "" {
		return ResponseResult{}, fmt.Errorf("provider auth token is not configured")
	}

	model := strings.TrimSpace(request.Model)
	if model == "" {
		model = strings.TrimSpace(cfg.Models.Default)
	}
	if model == "" {
		return ResponseResult{}, fmt.Errorf("provider default model is not configured")
	}
	if strings.TrimSpace(request.Input) == "" {
		return ResponseResult{}, fmt.Errorf("model input is required")
	}

	payload := map[string]any{
		"model": model,
		"input": request.Input,
	}
	if request.MaxOutputTokens > 0 {
		payload["max_output_tokens"] = request.MaxOutputTokens
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return ResponseResult{}, err
	}

	endpoint := strings.TrimRight(cfg.BaseURL, "/") + "/v1/responses"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ResponseResult{}, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := c.http.Do(httpRequest)
	if err != nil {
		return ResponseResult{}, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return ResponseResult{}, err
	}
	if response.StatusCode >= http.StatusBadRequest {
		detail := strings.TrimSpace(string(responseBody))
		if detail == "" {
			detail = response.Status
		}
		return ResponseResult{}, fmt.Errorf("%s", detail)
	}

	result, err := parseResponsesResult(responseBody)
	if err != nil {
		return ResponseResult{}, err
	}
	if strings.TrimSpace(result.OutputText) == "" {
		return ResponseResult{}, fmt.Errorf("empty output")
	}
	if strings.TrimSpace(result.Model) == "" {
		result.Model = model
	}
	result.APIStyle = APIStyleOpenAIResponses
	return result, nil
}

func parseResponsesResult(body []byte) (ResponseResult, error) {
	var payload struct {
		ID     string `json:"id"`
		Model  string `json:"model"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		OutputText string `json:"output_text"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ResponseResult{}, fmt.Errorf("invalid provider response: %w", err)
	}

	outputText := strings.TrimSpace(payload.OutputText)
	if outputText == "" {
		var parts []string
		for _, output := range payload.Output {
			for _, content := range output.Content {
				if text := strings.TrimSpace(content.Text); text != "" {
					parts = append(parts, text)
				}
			}
		}
		outputText = strings.Join(parts, "\n")
	}

	return ResponseResult{
		ResponseID: payload.ID,
		Model:      payload.Model,
		OutputText: outputText,
		Usage: Usage{
			InputTokens:  payload.Usage.InputTokens,
			OutputTokens: payload.Usage.OutputTokens,
			TotalTokens:  payload.Usage.TotalTokens,
		},
	}, nil
}
