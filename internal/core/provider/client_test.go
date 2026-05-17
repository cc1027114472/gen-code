package provider

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestClientCreateResponseRetriesEmptyOutput(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			_, _ = w.Write([]byte(`{"id":"resp_empty","model":"gpt-5.4-A","output":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"id":"resp_123",
			"model":"gpt-5.4-A",
			"output":[{"content":[{"type":"output_text","text":"hello after empty retry"}]}],
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

	originalSleep := providerSleep
	providerSleep = func(time.Duration) {}
	t.Cleanup(func() {
		providerSleep = originalSleep
	})

	result, err := NewClient(registry).CreateResponse(context.Background(), ResponseRequest{
		Input: "say hi",
	})
	require.NoError(t, err)
	require.Equal(t, "hello after empty retry", result.OutputText)
	require.Equal(t, 2, attempts)
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
	require.ErrorContains(t, err, "bad gateway")
}

func TestClientCreateResponseRejectsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":`))
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
	require.ErrorContains(t, err, "invalid provider response")
}

func TestClientCreateResponseRetriesTransientGatewayBodyError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid character 'e' looking for beginning of value","type":"bad_response_body","code":"bad_response_body"}}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"id":"resp_retry",
			"model":"gpt-5.4-A",
			"output":[{"content":[{"type":"output_text","text":"hello after retry"}]}],
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

	originalSleep := providerSleep
	providerSleep = func(time.Duration) {}
	t.Cleanup(func() {
		providerSleep = originalSleep
	})

	result, err := NewClient(registry).CreateResponse(context.Background(), ResponseRequest{
		Input: "say hi",
	})
	require.NoError(t, err)
	require.Equal(t, "hello after retry", result.OutputText)
	require.Equal(t, 2, attempts)
}

func TestClientCreateResponseDoesNotRetryNonTransientProviderError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid request","type":"invalid_request_error","code":"invalid_request"}}`))
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

	originalSleep := providerSleep
	providerSleep = func(time.Duration) {
		t.Fatalf("unexpected retry sleep")
	}
	t.Cleanup(func() {
		providerSleep = originalSleep
	})

	_, err := NewClient(registry).CreateResponse(context.Background(), ResponseRequest{
		Input: "say hi",
	})
	require.ErrorContains(t, err, "invalid request")
	require.Equal(t, 1, attempts)
}

func TestIsRetryableProviderErrorRecognizesTimeout(t *testing.T) {
	timeoutErr := &net.DNSError{IsTimeout: true, Err: "timeout", Name: "provider.test"}
	require.True(t, isRetryableProviderError(timeoutErr))
	require.True(t, isRetryableProviderError(context.DeadlineExceeded))
	require.False(t, isRetryableProviderError(fmt.Errorf("permanent failure")))
}
