package xhttpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestQueueFull ????????????
func TestQueueFull(t *testing.T) {
	cli := &client{cfg: Config{Workers: 1, QueueSize: 1}}
	pool := &Pool{
		queue:  make(chan job, 1),
		client: cli,
	}
	pool.queue <- job{}

	_, err := pool.Submit(context.Background(), Request{
		Name:   "overflow",
		Method: "GET",
		URL:    "http://example.com",
	})
	require.Error(t, err)
	require.True(t, IsKind(err, KindQueueFull))
}

// TestPoolWorkerResolvesFuture ????????????
func TestPoolWorkerResolvesFuture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "worker"})
	}))
	defer server.Close()

	cli := New(server.Client(), nil, Config{
		Workers:   1,
		QueueSize: 1,
	}).(*client)

	future, err := cli.pool.Submit(context.Background(), Request{
		Name:   "worker-test",
		Method: http.MethodGet,
		URL:    server.URL,
	})
	require.NoError(t, err)

	result, ok := future.WaitTimeout(time.Second)
	require.True(t, ok)
	require.NoError(t, result.Error)
	require.NotNil(t, result.Response)
	require.Equal(t, http.StatusOK, result.Response.StatusCode)
}
