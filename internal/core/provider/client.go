package provider

import (
	"bytes"
	"context"
	"errors"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const APIStyleOpenAIResponses = "openai-responses"

const (
	defaultResponseAttempts = 3
	responseRetryDelay      = 750 * time.Millisecond
)

var providerSleep = time.Sleep
var errEmptyProviderOutput = errors.New("empty output")

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

	var lastErr error
	for attempt := 1; attempt <= defaultResponseAttempts; attempt++ {
		result, err := c.createResponseAttempt(ctx, httpRequest, model)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if attempt == defaultResponseAttempts || !isRetryableProviderError(err) {
			return ResponseResult{}, err
		}
		providerSleep(responseRetryDelay)
	}
	return ResponseResult{}, lastErr
}

func (c *Client) createResponseAttempt(ctx context.Context, template *http.Request, model string) (ResponseResult, error) {
	request := template.Clone(ctx)
	if template.GetBody != nil {
		body, err := template.GetBody()
		if err != nil {
			return ResponseResult{}, err
		}
		request.Body = body
	}

	response, err := c.http.Do(request)
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
		return ResponseResult{}, &providerHTTPError{statusCode: response.StatusCode, detail: detail}
	}

	result, err := parseResponsesResult(responseBody)
	if err != nil {
		return ResponseResult{}, err
	}
	if strings.TrimSpace(result.OutputText) == "" {
		return ResponseResult{}, errEmptyProviderOutput
	}
	if strings.TrimSpace(result.Model) == "" {
		result.Model = model
	}
	result.APIStyle = APIStyleOpenAIResponses
	return result, nil
}

type providerHTTPError struct {
	statusCode int
	detail     string
}

func (e *providerHTTPError) Error() string {
	return e.detail
}

func isRetryableProviderError(err error) bool {
	if err == nil {
		return false
	}

	var httpErr *providerHTTPError
	if errors.As(err, &httpErr) {
		if httpErr.statusCode == http.StatusBadGateway || httpErr.statusCode == http.StatusServiceUnavailable || httpErr.statusCode == http.StatusGatewayTimeout {
			return true
		}
		detail := strings.ToLower(strings.TrimSpace(httpErr.detail))
		return strings.Contains(detail, `"code":"bad_response_body"`) || strings.Contains(detail, `"type":"bad_response_body"`) || strings.Contains(detail, `bad_response_body`)
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "eof") {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, errEmptyProviderOutput)
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
