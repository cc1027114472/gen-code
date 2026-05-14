package response

// ErrorBody 表示错误响应的输出结构。
type ErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// AppError 统一封装业务错误与 HTTP 状态码。
type AppError struct {
	HTTPStatus int
	Code       int
	Message    string
}

// Error 返回错误消息，满足 error 接口。
func (e AppError) Error() string {
	return e.Message
}
