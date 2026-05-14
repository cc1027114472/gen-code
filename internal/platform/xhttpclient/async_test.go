package xhttpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSubmitAndWait ????????????
func TestSubmitAndWait(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "alice"})
	}))
	defer server.Close()

	cli := New(server.Client(), nil, Config{
		Workers:   1,
		QueueSize: 1,
	}).(*client)

	future, err := cli.Submit(context.Background(), Request{
		Name:   "async-user",
		Method: http.MethodGet,
		URL:    server.URL,
	})
	require.NoError(t, err)
	result := future.Wait()
	require.NoError(t, result.Error)
	require.NotNil(t, result.Response)
	require.Equal(t, http.StatusOK, result.Response.StatusCode)
}

// TestSubmitBatch ????????????
func TestSubmitBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "alice"})
	}))
	defer server.Close()

	cli := New(server.Client(), nil, Config{
		Workers:   2,
		QueueSize: 2,
	}).(*client)

	futures, err := cli.SubmitBatch(context.Background(), []Request{
		{Name: "a", Method: http.MethodGet, URL: server.URL},
		{Name: "b", Method: http.MethodGet, URL: server.URL},
	})
	require.NoError(t, err)
	require.Len(t, futures, 2)

	for _, future := range futures {
		result := future.Wait()
		require.NoError(t, result.Error)
		require.NotNil(t, result.Response)
		require.Equal(t, http.StatusOK, result.Response.StatusCode)
	}
}

// TestSubmitBatchReturnsPartialFuturesWhenQueueIsFull ????????????
func TestSubmitBatchReturnsPartialFuturesWhenQueueIsFull(t *testing.T) {
	cli := &client{}
	cli.pool = &Pool{
		queue:  make(chan job, 1),
		client: cli,
	}

	futures, err := cli.SubmitBatch(context.Background(), []Request{
		{Name: "a", Method: http.MethodGet, URL: "http://example.com"},
		{Name: "b", Method: http.MethodGet, URL: "http://example.com"},
	})

	require.Error(t, err)
	require.True(t, IsKind(err, KindQueueFull))
	require.Len(t, futures, 1)
	require.NotNil(t, futures[0])
	require.Len(t, cli.pool.queue, 1)
}
