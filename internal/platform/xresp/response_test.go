package xresp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"llmtrace/internal/platform/xerror"
)

// TestWriteErrorUsesAppErrorStatusAndBody 用于验证业务错误会映射到对应状态码和响应体。
func TestWriteErrorUsesAppErrorStatusAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, rec := newTestContext()

	WriteError(c, xerror.NotFound(1004, "user not found"))

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.JSONEq(t, `{"code":1004,"message":"user not found"}`, rec.Body.String())
}

// TestWriteErrorMapsBadRequestByBusinessCode 用于验证请求错误编码会映射为 400。
func TestWriteErrorMapsBadRequestByBusinessCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, rec := newTestContext()

	WriteError(c, xerror.BadRequest(1001, "invalid request body"))

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.JSONEq(t, `{"code":1001,"message":"invalid request body"}`, rec.Body.String())
}

// TestWriteErrorFallsBackToInternalServerError 用于验证未知错误会回退为内部错误响应。
func TestWriteErrorFallsBackToInternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, rec := newTestContext()

	WriteError(c, errors.New("boom"))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.JSONEq(t, `{"code":500,"message":"internal server error"}`, rec.Body.String())
}

// TestCreatedKeepsSuccessContract 用于验证创建响应仍然遵循统一成功响应结构。
func TestCreatedKeepsSuccessContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, rec := newTestContext()

	Created(c, gin.H{"id": 1})

	require.Equal(t, http.StatusCreated, rec.Code)
	require.JSONEq(t, `{"code":0,"message":"ok","data":{"id":1}}`, rec.Body.String())
}

// newTestContext 用于创建响应测试所需的 Gin 上下文。
func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	return c, rec
}
