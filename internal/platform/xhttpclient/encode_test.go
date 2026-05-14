package xhttpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuildHTTPRequestEncodesJSONBody ????????????
func TestBuildHTTPRequestEncodesJSONBody(t *testing.T) {
	req, err := BuildHTTPRequest(context.Background(), Request{
		Method:   "post",
		URL:      "https://example.com/users",
		Query:    map[string]string{"page": "1"},
		Headers:  map[string]string{"X-Request-ID": "req-1"},
		JSONBody: map[string]any{"name": "alice"},
	})

	require.NoError(t, err)
	require.Equal(t, http.MethodPost, req.Method)
	require.Equal(t, "https://example.com/users?page=1", req.URL.String())
	require.Equal(t, ContentTypeJSON, req.Header.Get("Content-Type"))
	require.Equal(t, "req-1", req.Header.Get("X-Request-ID"))

	body, readErr := io.ReadAll(req.Body)
	require.NoError(t, readErr)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Equal(t, "alice", decoded["name"])
}

// TestBuildHTTPRequestEncodesFormBody ????????????
func TestBuildHTTPRequestEncodesFormBody(t *testing.T) {
	req, err := BuildHTTPRequest(context.Background(), Request{
		Method:   "POST",
		URL:      "https://example.com/forms",
		FormBody: map[string]string{"name": "alice", "role": "admin"},
	})

	require.NoError(t, err)
	require.Equal(t, ContentTypeForm, req.Header.Get("Content-Type"))

	body, readErr := io.ReadAll(req.Body)
	require.NoError(t, readErr)
	require.Contains(t, string(body), "name=alice")
	require.Contains(t, string(body), "role=admin")
}

// TestBuildUploadHTTPRequestEncodesMultipart ????????????
func TestBuildUploadHTTPRequestEncodesMultipart(t *testing.T) {
	req, err := BuildUploadHTTPRequest(context.Background(), UploadRequest{
		Request: Request{
			Method:   "POST",
			URL:      "https://example.com/upload",
			FormBody: map[string]string{"folder": "docs"},
		},
		Files: []UploadFile{
			{
				FieldName: "file",
				FileName:  "hello.txt",
				Reader:    bytes.NewBufferString("hello world"),
			},
		},
	})

	require.NoError(t, err)
	require.Contains(t, req.Header.Get("Content-Type"), ContentTypeMultipart)

	body, readErr := io.ReadAll(req.Body)
	require.NoError(t, readErr)
	bodyText := string(body)
	require.Contains(t, bodyText, `name="folder"`)
	require.Contains(t, bodyText, "docs")
	require.Contains(t, bodyText, `filename="hello.txt"`)
	require.Contains(t, bodyText, "hello world")
}

// TestBuildHTTPRequestPreservesRawBodyContentType ????????????
func TestBuildHTTPRequestPreservesRawBodyContentType(t *testing.T) {
	req, err := BuildHTTPRequest(context.Background(), Request{
		Method:      "PUT",
		URL:         "https://example.com/raw",
		RawBody:     []byte("payload"),
		ContentType: "text/plain",
	})

	require.NoError(t, err)
	require.Equal(t, "text/plain", req.Header.Get("Content-Type"))

	body, readErr := io.ReadAll(req.Body)
	require.NoError(t, readErr)
	require.Equal(t, "payload", string(body))
}

// TestEncodeJSONBodyWrapsMarshalError ????????????
func TestEncodeJSONBodyWrapsMarshalError(t *testing.T) {
	_, _, err := EncodeJSONBody(map[string]any{
		"broken": make(chan int),
	})

	require.Error(t, err)
	require.True(t, IsKind(err, KindEncode))
}

// TestBuildHTTPRequestRejectsInvalidRequest ????????????
func TestBuildHTTPRequestRejectsInvalidRequest(t *testing.T) {
	_, err := BuildHTTPRequest(context.Background(), Request{
		Method: "GET",
		URL:    "://bad",
	})

	require.Error(t, err)
	require.True(t, IsKind(err, KindValidation) || IsKind(err, KindBuildRequest))
}

// TestEncodeFormBodyProducesURLValues ????????????
func TestEncodeFormBodyProducesURLValues(t *testing.T) {
	body, contentType, err := EncodeFormBody(map[string]string{
		"name": "alice",
		"role": "admin user",
	})

	require.NoError(t, err)
	require.Equal(t, ContentTypeForm, contentType)
	require.True(t, strings.Contains(string(body), "name=alice"))
	require.True(t, strings.Contains(string(body), "role=admin+user"))
}
