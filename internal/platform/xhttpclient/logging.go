package xhttpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"llmtrace/internal/platform/xlog"
)

const (
	LogCallRequest      = "调用接口"
	LogCallResult       = "调用结果"
	LogCallError        = "调用异常"
	LogAsyncSubmit      = "提交异步请求"
	LogAsyncComplete    = "异步请求完成"
	LogAsyncSubmitError = "异步请求提交失败"
)

// logRequest ???????????
func logRequest(ctx context.Context, log xlog.Logger, cfg Config, req Request, finalURL string, headers map[string]string, bodyPreview string) {
	if log == nil {
		return
	}

	xlog.FromContext(ctx, log).Info(LogCallRequest,
		"name", req.Name,
		"method", req.NormalizedMethod(),
		"url", finalURL,
		"headers", headers,
		"params", req.QueryClone(),
		"body", truncate(bodyPreview, cfg.MaxRequestLogBody),
	)
}

// logResult ?????????????
func logResult(ctx context.Context, log xlog.Logger, cfg Config, req Request, finalURL string, resp *Response) {
	if log == nil || resp == nil {
		return
	}

	xlog.FromContext(ctx, log).Info(LogCallResult,
		"name", req.Name,
		"method", req.NormalizedMethod(),
		"url", finalURL,
		"status", resp.StatusCode,
		"duration_ms", resp.Duration.Milliseconds(),
		"result", truncate(string(resp.Body), cfg.MaxResponseLogBody),
	)
}

// logError ???????????
func logError(ctx context.Context, log xlog.Logger, cfg Config, req Request, finalURL string, statusCode int, err error, bodyPreview string, resultPreview string) {
	if log == nil {
		return
	}

	errorText := ""
	if err != nil {
		errorText = err.Error()
	}

	xlog.FromContext(ctx, log).Error(LogCallError,
		"name", req.Name,
		"method", req.NormalizedMethod(),
		"url", finalURL,
		"status", statusCode,
		"error", errorText,
		"body", truncate(bodyPreview, cfg.MaxRequestLogBody),
		"result", truncate(resultPreview, cfg.MaxResponseLogBody),
	)
}

// logAsyncSubmit ?????????????
func logAsyncSubmit(ctx context.Context, log xlog.Logger, req Request, queueSize int) {
	if log == nil {
		return
	}

	xlog.FromContext(ctx, log).Info(LogAsyncSubmit,
		"name", req.Name,
		"method", req.NormalizedMethod(),
		"url", req.URL,
		"queue_size", queueSize,
	)
}

// logAsyncComplete ?????????????
func logAsyncComplete(ctx context.Context, log xlog.Logger, req Request, resp *Response) {
	if log == nil || resp == nil {
		return
	}

	xlog.FromContext(ctx, log).Info(LogAsyncComplete,
		"name", req.Name,
		"method", req.NormalizedMethod(),
		"url", req.URL,
		"status", resp.StatusCode,
		"duration_ms", resp.Duration.Milliseconds(),
	)
}

// logAsyncSubmitError ???????????????
func logAsyncSubmitError(ctx context.Context, log xlog.Logger, req Request, err error) {
	if log == nil {
		return
	}

	errorText := ""
	if err != nil {
		errorText = err.Error()
	}

	xlog.FromContext(ctx, log).Error(LogAsyncSubmitError,
		"name", req.Name,
		"method", req.NormalizedMethod(),
		"url", req.URL,
		"error", errorText,
	)
}

// maskHeaders ??????????
func maskHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}

	masked := make(map[string]string, len(headers))
	for key, value := range headers {
		lower := strings.ToLower(key)
		switch {
		case lower == "authorization",
			lower == "proxy-authorization",
			lower == "cookie",
			lower == "set-cookie",
			lower == "x-api-key",
			strings.Contains(lower, "token"),
			strings.Contains(lower, "secret"),
			strings.Contains(lower, "password"):
			masked[key] = "***"
		default:
			masked[key] = value
		}
	}

	return masked
}

// previewRequestBody ????????????
func previewRequestBody(req Request) string {
	switch {
	case req.JSONBody != nil:
		payload, err := json.Marshal(req.JSONBody)
		if err != nil {
			return fmt.Sprintf("%v", req.JSONBody)
		}
		return string(payload)
	case len(req.FormBody) > 0:
		payload, _, err := EncodeFormBody(req.FormBody)
		if err != nil {
			return fmt.Sprintf("%v", req.FormBody)
		}
		return string(payload)
	case len(req.RawBody) > 0:
		return string(req.RawBody)
	default:
		return ""
	}
}

// truncate ????????????
func truncate(value string, max int) string {
	return truncateString(value, max)
}
