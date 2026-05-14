package xerror

import (
	"errors"
)

// Error 表示带业务编码和底层原因的应用错误。
type Error struct {
	Code    int
	Message string
	Cause   error
}

// Error 返回错误消息文本。
func (e Error) Error() string {
	return e.Message
}

// Unwrap 返回被包装的底层错误。
func (e Error) Unwrap() error {
	return e.Cause
}

// New 用于创建新的业务错误。
func New(code int, message string) Error {
	return Error{
		Code:    code,
		Message: message,
	}
}

// Wrap 用于为底层错误追加业务错误信息。
func Wrap(err error, code int, message string) Error {
	return Error{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// BadRequest 用于构造请求参数错误。
func BadRequest(code int, message string) Error {
	return New(code, message)
}

// Unauthorized 用于构造未授权错误。
func Unauthorized(code int, message string) Error {
	return New(code, message)
}

// NotFound 用于构造资源不存在错误。
func NotFound(code int, message string) Error {
	return New(code, message)
}

// Conflict 用于构造资源冲突错误。
func Conflict(code int, message string) Error {
	return New(code, message)
}

// Forbidden 用于构造禁止访问错误。
func Forbidden(code int, message string) Error {
	return New(code, message)
}

// Internal 用于构造内部错误。
func Internal(code int, message string) Error {
	return New(code, message)
}

// As 用于从错误链中提取业务错误。
func As(err error) (Error, bool) {
	var xerr Error
	if errors.As(err, &xerr) {
		return xerr, true
	}

	return Error{}, false
}

// CodeOf 用于获取错误对应的业务编码。
func CodeOf(err error) int {
	if xerr, ok := As(err); ok {
		return xerr.Code
	}

	return 0
}

// Message 用于获取错误消息文本。
func Message(err error) string {
	if xerr, ok := As(err); ok {
		return xerr.Message
	}

	if err == nil {
		return ""
	}

	return err.Error()
}

// MessageOf 用于获取错误消息文本。
func MessageOf(err error) string {
	return Message(err)
}
