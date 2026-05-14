package xhttpclient

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMaskHeaders ????????????
func TestMaskHeaders(t *testing.T) {
	headers := maskHeaders(map[string]string{
		"Authorization":       "Bearer 123",
		"Cookie":              "a=b",
		"Set-Cookie":          "sid=abc",
		"X-API-Key":           "secret-key",
		"X-Access-Token":      "token-value",
		"X-Custom-Secret":     "hidden",
		"X-Test":              "ok",
		"Proxy-Authorization": "Basic abc",
	})

	require.Equal(t, "***", headers["Authorization"])
	require.Equal(t, "***", headers["Cookie"])
	require.Equal(t, "***", headers["Set-Cookie"])
	require.Equal(t, "***", headers["X-API-Key"])
	require.Equal(t, "***", headers["X-Access-Token"])
	require.Equal(t, "***", headers["X-Custom-Secret"])
	require.Equal(t, "***", headers["Proxy-Authorization"])
	require.Equal(t, "ok", headers["X-Test"])
}

// TestTruncate ????????????
func TestTruncate(t *testing.T) {
	require.Equal(t, "abc...(已截断)", truncate("abcdef", 3))
	require.Equal(t, "abc", truncate("abc", 10))
	require.Equal(t, "", truncate("abcdef", 0))
}

// TestTruncateSupportsUTF8 ????????????
func TestTruncateSupportsUTF8(t *testing.T) {
	require.Equal(t, "你好...(已截断)", truncate("你好世界", 2))
}

// TestWrapQueueFullUsesChineseMessage ????????????
func TestWrapQueueFullUsesChineseMessage(t *testing.T) {
	err := WrapQueueFull()

	require.Error(t, err)
	require.True(t, IsKind(err, KindQueueFull))
	require.Contains(t, err.Error(), "op=request.submit")
	require.Contains(t, err.Error(), "kind=queue_full")
	require.Contains(t, err.Error(), "msg=异步请求队列已满")
}

// TestWrapStatusUsesChineseMessageAndTruncation ????????????
func TestWrapStatusUsesChineseMessageAndTruncation(t *testing.T) {
	body := "响应内容过长需要被截断"
	err := WrapStatus("request.status_check", 502, []byte(body))

	require.Error(t, err)
	require.True(t, IsKind(err, KindHTTPStatus))
	require.Equal(t, 502, StatusCodeOf(err))
	require.Contains(t, err.Error(), "status=502")
	require.Contains(t, err.Error(), "msg=HTTP状态码异常: 502")
	require.Contains(t, err.Error(), "body=响应内容过长需要被截断")
}
