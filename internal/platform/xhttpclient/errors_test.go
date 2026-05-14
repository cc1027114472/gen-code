package xhttpclient

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWrapPreservesKindAndCause ????????????
func TestWrapPreservesKindAndCause(t *testing.T) {
	cause := errors.New("boom")

	err := Wrap(KindExecute, "request.do", cause)

	require.Error(t, err)
	require.True(t, IsKind(err, KindExecute))
	require.ErrorIs(t, err, cause)
}

// TestWrapStatusCarriesStatusCode ????????????
func TestWrapStatusCarriesStatusCode(t *testing.T) {
	err := WrapStatus("request.do", 503, []byte("service unavailable"))

	require.True(t, IsKind(err, KindHTTPStatus))
	require.Equal(t, 503, StatusCodeOf(err))
	require.Contains(t, err.Error(), "503")
	require.Contains(t, err.Error(), "service unavailable")
}
