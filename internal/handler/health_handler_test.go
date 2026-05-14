package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestHealthzHandler 验证健康检查接口返回成功响应。
func TestHealthzHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	h := NewHealthHandler()
	r.GET("/healthz", h.Healthz)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"code":0`)
	require.Contains(t, rec.Body.String(), `"status":"ok"`)
}
