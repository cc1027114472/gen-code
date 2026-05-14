package response

// SuccessBody 表示成功响应的统一结构。
type SuccessBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// Success 根据业务数据生成标准成功响应体。
func Success(data any) SuccessBody {
	return SuccessBody{
		Code:    0,
		Message: "ok",
		Data:    data,
	}
}
