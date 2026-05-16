package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ProbeResult captures a lightweight provider connectivity and contract inference result.
type ProbeResult struct {
	Kind              Kind           `json:"kind"`
	Reachable         bool           `json:"reachable"`
	PreferredAPIStyle string         `json:"preferred_api_style"`
	Message           string         `json:"message"`
	Details           map[string]any `json:"details,omitempty"`
}

// Prober performs lightweight provider HTTP probes without sending full model workloads.
type Prober struct {
	client http.Client
}

// NewProber constructs a provider prober with a conservative timeout.
func NewProber() *Prober {
	return &Prober{
		client: http.Client{Timeout: 20 * time.Second},
	}
}

// Probe inspects a provider config and infers the most likely usable API style.
func (p *Prober) Probe(ctx context.Context, cfg Config) (ProbeResult, error) {
	result := ProbeResult{
		Kind:              cfg.Kind,
		Reachable:         false,
		PreferredAPIStyle: "unknown",
		Details:           map[string]any{},
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if !cfg.Enabled {
		result.Message = "provider disabled"
		return result, nil
	}
	if baseURL == "" {
		result.Message = "provider base URL is not configured"
		return result, nil
	}
	if strings.TrimSpace(cfg.AuthToken) == "" {
		result.Message = "provider auth token is not configured"
		return result, nil
	}

	model, err := p.fetchModelsOpenAI(ctx, baseURL, cfg.AuthToken)
	if err == nil {
		result.Reachable = true
		result.Details["openai_models"] = true
		result.Details["model_lookup"] = model
		if supported, ok := model["supported_endpoint_types"].([]any); ok {
			styles := make([]string, 0, len(supported))
			for _, item := range supported {
				if value, ok := item.(string); ok {
					styles = append(styles, value)
				}
			}
			result.Details["supported_endpoint_types"] = styles
			if containsString(styles, "openai") {
				result.PreferredAPIStyle = "openai-responses"
				result.Message = "provider reachable; model advertises openai compatibility"
				return result, nil
			}
		}
	}

	if anthropicOK, anthropicMessage := p.probeAnthropicModels(ctx, baseURL, cfg.AuthToken); anthropicOK {
		result.Reachable = true
		result.Details["anthropic_models"] = true
		if result.PreferredAPIStyle == "unknown" {
			result.PreferredAPIStyle = "anthropic"
		}
		result.Message = anthropicMessage
		return result, nil
	}

	result.Message = "provider probe did not find a supported contract"
	return result, nil
}

func (p *Prober) fetchModelsOpenAI(ctx context.Context, baseURL string, token string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("openai models probe failed: %s", resp.Status)
	}

	var payload struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Data) == 0 {
		return map[string]any{}, nil
	}
	return payload.Data[0], nil
}

func (p *Prober) probeAnthropicModels(ctx context.Context, baseURL string, token string) (bool, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return false, "anthropic models probe could not be created"
	}
	req.Header.Set("x-api-key", token)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return false, "anthropic models probe failed"
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return false, fmt.Sprintf("anthropic models probe returned %s", resp.Status)
	}
	return true, "provider reachable; anthropic-style models endpoint responds"
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}
