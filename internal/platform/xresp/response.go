package xresp

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"llmtrace/internal/platform/xerror"
)

// SuccessBody 表示成功响应体结构。
type SuccessBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// ErrorBody 表示错误响应体结构。
type ErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Success 用于构造成功响应体。
func Success(data any) SuccessBody {
	return SuccessBody{
		Code:    0,
		Message: "ok",
		Data:    data,
	}
}

// OK 用于输出 HTTP 200 成功响应。
func OK(c *gin.Context, data any) {
	WriteSuccess(c, http.StatusOK, data)
}

// Created 用于输出 HTTP 201 成功响应。
func Created(c *gin.Context, data any) {
	WriteSuccess(c, http.StatusCreated, data)
}

// WriteSuccess 用于输出统一成功响应。
func WriteSuccess(c *gin.Context, httpStatus int, data any) {
	c.JSON(httpStatus, Success(data))
}

// BadRequest 用于输出请求参数错误响应。
func BadRequest(c *gin.Context, code int, message string) {
	WriteError(c, xerror.BadRequest(code, message))
}

// NotFound 用于输出资源不存在错误响应。
func NotFound(c *gin.Context, code int, message string) {
	WriteError(c, xerror.NotFound(code, message))
}

// Conflict 用于输出资源冲突错误响应。
func Conflict(c *gin.Context, code int, message string) {
	WriteError(c, xerror.Conflict(code, message))
}

// WriteError 用于输出统一错误响应。
func WriteError(c *gin.Context, err error) {
	if err == nil {
		writeInternal(c)
		return
	}

	code := xerror.CodeOf(err)
	if code == 0 {
		writeInternal(c)
		return
	}

	c.JSON(statusOf(code), ErrorBody{
		Code:    code,
		Message: xerror.MessageOf(err),
	})
}

// statusOf 用于根据业务编码映射 HTTP 状态码。
func statusOf(code int) int {
	switch code {
	case 1001, 1003:
		return http.StatusBadRequest
	case 1002:
		return http.StatusConflict
	case 1004:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// writeInternal 用于输出默认内部错误响应。
func writeInternal(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, ErrorBody{
		Code:    http.StatusInternalServerError,
		Message: "internal server error",
	})
}
