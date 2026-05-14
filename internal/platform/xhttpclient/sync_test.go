package xhttpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetJSON ????????????
func TestGetJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "alice"})
	}))
	defer server.Close()

	cli := New(server.Client(), nil, DefaultConfig())
	var out map[string]string
	err := cli.GetJSON(context.Background(), Request{
		Name:   "get-user",
		Method: http.MethodGet,
		URL:    server.URL,
	}, &out)
	require.NoError(t, err)
	require.Equal(t, "alice", out["name"])
}

// TestDownload ????????????
func TestDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	cli := New(server.Client(), nil, DefaultConfig())
	path := filepath.Join(t.TempDir(), "a.txt")
	result, err := cli.Download(context.Background(), DownloadRequest{
		Request: Request{
			Name:   "download-file",
			Method: http.MethodGet,
			URL:    server.URL,
		},
		SavePath: path,
	})
	require.NoError(t, err)
	require.Equal(t, int64(5), result.Size)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "hello", string(data))
}
