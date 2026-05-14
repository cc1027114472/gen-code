package xhttpclient

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestRequestValidateRejectsConflictingBodies ????????????
func TestRequestValidateRejectsConflictingBodies(t *testing.T) {
	req := Request{
		Method:   "POST",
		URL:      "https://example.com",
		JSONBody: map[string]any{"name": "alice"},
		FormBody: map[string]string{"name": "alice"},
	}

	err := req.Validate()

	require.Error(t, err)
	require.True(t, IsKind(err, KindValidation))
	require.ErrorIs(t, err, errConflictingBodies)
}

// TestRequestValidateRejectsRelativeURL ????????????
func TestRequestValidateRejectsRelativeURL(t *testing.T) {
	req := Request{
		Method: "GET",
		URL:    "/relative",
	}

	err := req.Validate()

	require.Error(t, err)
	require.True(t, IsKind(err, KindValidation))
	require.ErrorIs(t, err, errInvalidURL)
}

// TestUploadRequestValidateAllowsFormAndFiles ????????????
func TestUploadRequestValidateAllowsFormAndFiles(t *testing.T) {
	req := UploadRequest{
		Request: Request{
			Method:   "post",
			URL:      "https://example.com/upload",
			FormBody: map[string]string{"folder": "docs"},
		},
		Files: []UploadFile{
			{
				FieldName: "file",
				FileName:  "a.txt",
				Reader:    bytes.NewBufferString("hello"),
			},
		},
	}

	err := req.Validate()

	require.NoError(t, err)
}

// TestDownloadRequestValidateRequiresSavePath ????????????
func TestDownloadRequestValidateRequiresSavePath(t *testing.T) {
	req := DownloadRequest{
		Request: Request{
			Method: "GET",
			URL:    "https://example.com/file",
		},
	}

	err := req.Validate()

	require.Error(t, err)
	require.True(t, IsKind(err, KindValidation))
	require.ErrorIs(t, err, errMissingSavePath)
}

// TestBuildDownloadHTTPRequestUsesBaseRequest ????????????
func TestBuildDownloadHTTPRequestUsesBaseRequest(t *testing.T) {
	httpReq, err := BuildDownloadHTTPRequest(context.Background(), DownloadRequest{
		Request: Request{
			Method: "GET",
			URL:    "https://example.com/file",
			Query:  map[string]string{"part": "1"},
			Headers: map[string]string{
				"Accept": "application/octet-stream",
			},
		},
		SavePath: "download.bin",
	})

	require.NoError(t, err)
	require.Equal(t, http.MethodGet, httpReq.Method)
	require.Equal(t, "https://example.com/file?part=1", httpReq.URL.String())
	require.Equal(t, "application/octet-stream", httpReq.Header.Get("Accept"))
	require.Nil(t, httpReq.Body)
}

// TestBuildDownloadHTTPRequestRejectsMissingSavePath ????????????
func TestBuildDownloadHTTPRequestRejectsMissingSavePath(t *testing.T) {
	_, err := BuildDownloadHTTPRequest(context.Background(), DownloadRequest{
		Request: Request{
			Method: "GET",
			URL:    "https://example.com/file",
		},
	})

	require.Error(t, err)
	require.True(t, IsKind(err, KindValidation))
	require.ErrorIs(t, err, errMissingSavePath)
}

// TestRequestEffectiveTimeoutAndRetryUseConfigDefaults ????????????
func TestRequestEffectiveTimeoutAndRetryUseConfigDefaults(t *testing.T) {
	req := Request{
		Method: "GET",
		URL:    "https://example.com",
	}

	cfg := Config{
		Timeout: 3 * time.Second,
		Retry:   2,
	}

	require.Equal(t, 3*time.Second, req.EffectiveTimeout(cfg))
	require.Equal(t, 2, req.EffectiveRetry(cfg))
}
