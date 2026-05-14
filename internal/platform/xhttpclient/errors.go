package xhttpclient

import (
	"errors"
	"fmt"
	"strings"
)

type ErrorKind string

const (
	KindValidation   ErrorKind = "validation"
	KindQueueFull    ErrorKind = "queue_full"
	KindBuildRequest ErrorKind = "build_request"
	KindEncode       ErrorKind = "encode"
	KindExecute      ErrorKind = "execute"
	KindTimeout      ErrorKind = "timeout"
	KindDecode       ErrorKind = "decode"
	KindHTTPStatus   ErrorKind = "http_status"
	KindFileIO       ErrorKind = "file_io"
)

var (
	errMissingMethod                = errors.New("request method is required")
	errMissingURL                   = errors.New("request url is required")
	errInvalidURL                   = errors.New("request url must be absolute")
	errInvalidTimeout               = errors.New("request timeout must be greater than or equal to zero")
	errInvalidRetry                 = errors.New("request retry must be greater than or equal to zero")
	errConflictingBodies            = errors.New("request body types are mutually exclusive")
	errJSONContentTypeMismatch      = errors.New("json body requires application/json content type")
	errFormContentTypeMismatch      = errors.New("form body requires application/x-www-form-urlencoded content type")
	errMultipartContentTypeMismatch = errors.New("upload request requires multipart/form-data content type")
	errUploadBodyNotSupported       = errors.New("upload request does not support json or raw body")
	errMissingUploadFiles           = errors.New("upload request requires at least one file")
	errMissingUploadFieldName       = errors.New("upload file field name is required")
	errMissingUploadFileName        = errors.New("upload file name is required")
	errMissingUploadReader          = errors.New("upload file reader is required")
	errMissingSavePath              = errors.New("download save path is required")
	errNilResponse                  = errors.New("response is nil")
	errNilHTTPResponse              = errors.New("http response is nil")
)

// Error ?? HTTP ???????????
type Error struct {
	Kind       ErrorKind
	Op         string
	StatusCode int
	Message    string
	Body       string
	Err        error
}

// Error ?????????
func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	parts := make([]string, 0, 5)
	if e.Op != "" {
		parts = append(parts, fmt.Sprintf("op=%s", e.Op))
	}
	if e.Kind != "" {
		parts = append(parts, fmt.Sprintf("kind=%s", e.Kind))
	}
	if e.StatusCode > 0 {
		parts = append(parts, fmt.Sprintf("status=%d", e.StatusCode))
	}
	if e.Message != "" {
		parts = append(parts, fmt.Sprintf("msg=%s", e.Message))
	} else if e.Err != nil {
		parts = append(parts, fmt.Sprintf("msg=%s", e.Err.Error()))
	}
	if e.Body != "" {
		parts = append(parts, fmt.Sprintf("body=%s", e.Body))
	}

	return strings.Join(parts, " ")
}

// Unwrap ???????
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Wrap ????????????
func Wrap(kind ErrorKind, op string, err error) error {
	if err == nil {
		return nil
	}

	return &Error{
		Kind: kind,
		Op:   op,
		Err:  err,
	}
}

// NewError ????????????
func NewError(kind ErrorKind, op string, message string) error {
	return &Error{
		Kind:    kind,
		Op:      op,
		Message: strings.TrimSpace(message),
	}
}

// WrapBuildError ???????????
func WrapBuildError(err error) error {
	return Wrap(KindBuildRequest, "request.build", err)
}

// WrapExecuteError ???????????
func WrapExecuteError(err error) error {
	return Wrap(KindExecute, "request.execute", err)
}

// WrapDecodeError ???????????
func WrapDecodeError(err error) error {
	return Wrap(KindDecode, "response.decode", err)
}

// WrapFileError ???????????
func WrapFileError(err error) error {
	return Wrap(KindFileIO, "response.save_file", err)
}

// WrapStatus ???? HTTP ??????
func WrapStatus(op string, statusCode int, body []byte) error {
	return &Error{
		Kind:       KindHTTPStatus,
		Op:         op,
		StatusCode: statusCode,
		Message:    fmt.Sprintf("HTTP状态码异常: %d", statusCode),
		Body:       strings.TrimSpace(truncateString(string(body), 256)),
	}
}

// WrapStatusError ????????????
func WrapStatusError(statusCode int, body string) error {
	return WrapStatus("request.status_check", statusCode, []byte(body))
}

// WrapQueueFull ?????????????
func WrapQueueFull() error {
	return NewError(KindQueueFull, "request.submit", "异步请求队列已满")
}

// KindOf ?????????
func KindOf(err error) ErrorKind {
	var target *Error
	if errors.As(err, &target) {
		return target.Kind
	}
	return ""
}

// StatusCodeOf ????????????
func StatusCodeOf(err error) int {
	var target *Error
	if errors.As(err, &target) {
		return target.StatusCode
	}
	return 0
}

// IsKind ???????????????
func IsKind(err error, kind ErrorKind) bool {
	return KindOf(err) == kind
}

// truncateString ?????????????
func truncateString(value string, max int) string {
	if max <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "...(已截断)"
}
