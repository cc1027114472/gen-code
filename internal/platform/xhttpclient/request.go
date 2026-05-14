package xhttpclient

import (
	"io"
	"net/url"
	"strings"
	"time"
)

const (
	ContentTypeJSON      = "application/json"
	ContentTypeForm      = "application/x-www-form-urlencoded"
	ContentTypeMultipart = "multipart/form-data"
)

// Request ???? HTTP ?????
type Request struct {
	Name        string
	Method      string
	URL         string
	Query       map[string]string
	Headers     map[string]string
	JSONBody    any
	FormBody    map[string]string
	RawBody     []byte
	ContentType string
	Timeout     time.Duration
	Retry       int
	Metadata    map[string]string
}

// UploadRequest ???????????
type UploadRequest struct {
	Request
	Files []UploadFile
}

// DownloadRequest ???????????
type DownloadRequest struct {
	Request
	SavePath string
}

// UploadFile ????????????
type UploadFile struct {
	FieldName string
	FileName  string
	Reader    io.Reader
	Size      int64
}

// Validate ?????????????
func (r Request) Validate() error {
	if err := validateMethodAndURL(r.Method, r.URL); err != nil {
		return Wrap(KindValidation, "request.validate", err)
	}
	if r.Timeout < 0 {
		return Wrap(KindValidation, "request.validate", errInvalidTimeout)
	}
	if r.Retry < 0 {
		return Wrap(KindValidation, "request.validate", errInvalidRetry)
	}

	if bodyKindCount(r.JSONBody, r.FormBody, r.RawBody) > 1 {
		return Wrap(KindValidation, "request.validate", errConflictingBodies)
	}

	if r.JSONBody != nil && r.ContentType != "" && !strings.Contains(strings.ToLower(r.ContentType), ContentTypeJSON) {
		return Wrap(KindValidation, "request.validate", errJSONContentTypeMismatch)
	}
	if len(r.FormBody) > 0 && r.ContentType != "" && !strings.Contains(strings.ToLower(r.ContentType), ContentTypeForm) {
		return Wrap(KindValidation, "request.validate", errFormContentTypeMismatch)
	}

	return nil
}

// NormalizedMethod ????????? HTTP ???
func (r Request) NormalizedMethod() string {
	return strings.ToUpper(strings.TrimSpace(r.Method))
}

// EffectiveTimeout ????????????????
func (r Request) EffectiveTimeout(cfg Config) time.Duration {
	if r.Timeout > 0 {
		return r.Timeout
	}
	return cfg.Normalize().Timeout
}

// EffectiveRetry ????????????????
func (r Request) EffectiveRetry(cfg Config) int {
	if r.Retry > 0 {
		return r.Retry
	}
	return cfg.Normalize().Retry
}

// HeaderClone ???????????????
func (r Request) HeaderClone() map[string]string {
	return cloneStringMap(r.Headers)
}

// QueryClone ????????????
func (r Request) QueryClone() map[string]string {
	return cloneStringMap(r.Query)
}

// MetadataClone ???????????
func (r Request) MetadataClone() map[string]string {
	return cloneStringMap(r.Metadata)
}

// Validate ?????????????
func (r UploadRequest) Validate() error {
	if err := validateMethodAndURL(r.Method, r.URL); err != nil {
		return Wrap(KindValidation, "upload_request.validate", err)
	}
	if r.Timeout < 0 {
		return Wrap(KindValidation, "upload_request.validate", errInvalidTimeout)
	}
	if r.Retry < 0 {
		return Wrap(KindValidation, "upload_request.validate", errInvalidRetry)
	}
	if r.JSONBody != nil || len(r.RawBody) > 0 {
		return Wrap(KindValidation, "upload_request.validate", errUploadBodyNotSupported)
	}
	if len(r.Files) == 0 {
		return Wrap(KindValidation, "upload_request.validate", errMissingUploadFiles)
	}
	for _, file := range r.Files {
		if strings.TrimSpace(file.FieldName) == "" {
			return Wrap(KindValidation, "upload_request.validate", errMissingUploadFieldName)
		}
		if strings.TrimSpace(file.FileName) == "" {
			return Wrap(KindValidation, "upload_request.validate", errMissingUploadFileName)
		}
		if file.Reader == nil {
			return Wrap(KindValidation, "upload_request.validate", errMissingUploadReader)
		}
	}
	if r.ContentType != "" && !strings.Contains(strings.ToLower(r.ContentType), ContentTypeMultipart) {
		return Wrap(KindValidation, "upload_request.validate", errMultipartContentTypeMismatch)
	}

	return nil
}

// Validate ?????????????
func (r DownloadRequest) Validate() error {
	if err := r.Request.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.SavePath) == "" {
		return Wrap(KindValidation, "download_request.validate", errMissingSavePath)
	}
	return nil
}

// buildURL ????????????? URL?
func buildURL(rawURL string, query map[string]string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", Wrap(KindBuildRequest, "request.build_url", err)
	}

	values := parsed.Query()
	for key, value := range query {
		values.Set(key, value)
	}
	parsed.RawQuery = values.Encode()

	return parsed.String(), nil
}

// validateMethodAndURL ????????? URL?
func validateMethodAndURL(method string, rawURL string) error {
	if strings.TrimSpace(method) == "" {
		return errMissingMethod
	}
	if strings.TrimSpace(rawURL) == "" {
		return errMissingURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return errInvalidURL
	}

	return nil
}

// bodyKindCount ?????????????
func bodyKindCount(jsonBody any, formBody map[string]string, rawBody []byte) int {
	count := 0
	if jsonBody != nil {
		count++
	}
	if len(formBody) > 0 {
		count++
	}
	if len(rawBody) > 0 {
		count++
	}
	return count
}

// cloneStringMap ??????????
func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}

	return dst
}
