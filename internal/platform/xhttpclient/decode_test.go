package xhttpclient

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestReadResponseCopiesBodyAndHeaders ????????????
func TestReadResponseCopiesBodyAndHeaders(t *testing.T) {
	httpResp := &http.Response{
		StatusCode: 201,
		Header: http.Header{
			"X-Test": []string{"1"},
		},
		Body: io.NopCloser(strings.NewReader(`{"ok":true}`)),
	}

	resp, err := ReadResponse(httpResp, 25*time.Millisecond)

	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode)
	require.Equal(t, []string{"1"}, resp.Headers["X-Test"])
	require.JSONEq(t, `{"ok":true}`, string(resp.Body))
	require.Equal(t, 25*time.Millisecond, resp.Duration)
}

// TestDecodeJSONResponseUnmarshalsPayload ????????????
func TestDecodeJSONResponseUnmarshalsPayload(t *testing.T) {
	resp := &Response{
		StatusCode: 200,
		Body:       []byte(`{"name":"alice"}`),
	}

	var out struct {
		Name string `json:"name"`
	}
	err := DecodeJSONResponse(resp, &out)

	require.NoError(t, err)
	require.Equal(t, "alice", out.Name)
}

// TestDecodeJSONBodyWrapsInvalidJSON ????????????
func TestDecodeJSONBodyWrapsInvalidJSON(t *testing.T) {
	var out struct {
		Name string `json:"name"`
	}

	err := DecodeJSONBody([]byte(`{"name":`), &out)

	require.Error(t, err)
	require.True(t, IsKind(err, KindDecode))
}

// TestSaveResponseBodyWritesFile ????????????
func TestSaveResponseBodyWritesFile(t *testing.T) {
	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "nested", "download.txt")
	resp := &Response{
		StatusCode: 200,
		Body:       []byte("hello"),
		Duration:   50 * time.Millisecond,
	}

	result, err := SaveResponseBody(resp, savePath)

	require.NoError(t, err)
	require.Equal(t, savePath, result.SavePath)
	require.EqualValues(t, 5, result.Size)
	require.Equal(t, 200, result.StatusCode)

	content, readErr := os.ReadFile(savePath)
	require.NoError(t, readErr)
	require.Equal(t, "hello", string(content))
}

// TestSaveHTTPResponseStreamsToDisk ????????????
func TestSaveHTTPResponseStreamsToDisk(t *testing.T) {
	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "stream.bin")
	httpResp := &http.Response{
		StatusCode: 206,
		Body:       io.NopCloser(strings.NewReader("payload")),
	}

	result, err := SaveHTTPResponse(httpResp, savePath, 100*time.Millisecond)

	require.NoError(t, err)
	require.Equal(t, 206, result.StatusCode)
	require.EqualValues(t, 7, result.Size)

	content, readErr := os.ReadFile(savePath)
	require.NoError(t, readErr)
	require.Equal(t, "payload", string(content))
}
