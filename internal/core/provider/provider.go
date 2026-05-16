package provider

import "strings"

// Kind identifies a model provider family.
type Kind string

const (
	Anthropic Kind = "anthropic"
	OpenAI    Kind = "openai"
	Gemini    Kind = "gemini"
)

// Models stores model aliases exposed to the rest of the runtime.
type Models struct {
	Default string `json:"default"`
	Haiku   string `json:"haiku"`
	Sonnet  string `json:"sonnet"`
	Opus    string `json:"opus"`
}

// Config describes a provider without leaking env-key concerns into core.
type Config struct {
	Kind      Kind   `json:"kind"`
	Enabled   bool   `json:"enabled"`
	BaseURL   string `json:"base_url"`
	AuthToken string `json:"-"`
	Models    Models `json:"models"`
}

// Descriptor is the safe runtime view surfaced to future callers and UIs.
type Descriptor struct {
	Kind          Kind   `json:"kind"`
	Enabled       bool   `json:"enabled"`
	BaseURL       string `json:"base_url"`
	DefaultModel  string `json:"default_model"`
	HasAuthToken  bool   `json:"has_auth_token"`
	SupportsChat  bool   `json:"supports_chat"`
	SupportsTools bool   `json:"supports_tools"`
}

// NormalizeKind canonicalizes provider kind input.
func NormalizeKind(value string) Kind {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(Anthropic):
		return Anthropic
	case string(OpenAI):
		return OpenAI
	case string(Gemini):
		return Gemini
	default:
		return Kind(strings.ToLower(strings.TrimSpace(value)))
	}
}

// Descriptor returns the safe summary form of the provider config.
func (c Config) Descriptor() Descriptor {
	return Descriptor{
		Kind:          c.Kind,
		Enabled:       c.Enabled,
		BaseURL:       c.BaseURL,
		DefaultModel:  c.Models.Default,
		HasAuthToken:  c.AuthToken != "",
		SupportsChat:  c.Enabled,
		SupportsTools: false,
	}
}
