package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientCreateResponseParsesOutputText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/responses", r.URL.Path)
		require.Equal(t, "Bearer token-1", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{
			"id":"resp_123",
			"model":"gpt-5.4-A",
			"output":[{"content":[{"type":"output_text","text":"hello from gateway"}]}],
			"usage":{"input_tokens":12,"output_tokens":8,"total_tokens":20}
		}`))
	}))
	defer server.Close()

	registry := NewRegistry(Anthropic)
	registry.Register(Config{
		Kind:      Anthropic,
		Enabled:   true,
		BaseURL:   server.URL,
		AuthToken: "token-1",
		Models: Models{
			Default: "gpt-5.4-A",
		},
	})

	result, err := NewClient(registry).CreateResponse(context.Background(), ResponseRequest{
		Input: "say hi",
	})
	require.NoError(t, err)
	require.Equal(t, "resp_123", result.ResponseID)
	require.Equal(t, "gpt-5.4-A", result.Model)
	require.Equal(t, "hello from gateway", result.OutputText)
	require.Equal(t, APIStyleOpenAIResponses, result.APIStyle)
	require.Equal(t, 20, result.Usage.TotalTokens)
}

func TestClientCreateResponseRejectsEmptyOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"resp_empty","model":"gpt-5.4-A","output":[]}`))
	}))
	defer server.Close()

	registry := NewRegistry(Anthropic)
	registry.Register(Config{
		Kind:      Anthropic,
		Enabled:   true,
		BaseURL:   server.URL,
		AuthToken: "token-1",
		Models: Models{
			Default: "gpt-5.4-A",
		},
	})

	_, err := NewClient(registry).CreateResponse(context.Background(), ResponseRequest{
		Input: "say hi",
	})
	require.ErrorContains(t, err, "empty output")
}

func TestClientCreateResponseHandlesProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"bad gateway"}`, http.StatusBadGateway)
	}))
	defer server.Close()

	registry := NewRegistry(Anthropic)
	registry.Register(Config{
		Kind:      Anthropic,
		Enabled:   true,
		BaseURL:   server.URL,
		AuthToken: "token-1",
		Models: Models{
			Default: "gpt-5.4-A",
		},
	})

	_, err := NewClient(registry).CreateResponse(context.Background(), ResponseRequest{
		Input: "say hi",
	})
	require.ErrorContains(t, err, "provider error")
}
