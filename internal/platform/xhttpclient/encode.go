package xhttpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// BuildHTTPRequest ?????? HTTP ?????
func BuildHTTPRequest(ctx context.Context, req Request) (*http.Request, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	finalURL, err := buildURL(req.URL, req.Query)
	if err != nil {
		return nil, err
	}

	body, contentType, err := encodeRequestBody(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.NormalizedMethod(), finalURL, body)
	if err != nil {
		return nil, Wrap(KindBuildRequest, "request.new_http_request", err)
	}

	applyHeaders(httpReq, req.Headers)
	if contentType != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	return httpReq, nil
}

// BuildUploadHTTPRequest ???????????
func BuildUploadHTTPRequest(ctx context.Context, req UploadRequest) (*http.Request, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	finalURL, err := buildURL(req.URL, req.Query)
	if err != nil {
		return nil, err
	}

	body, contentType, err := EncodeMultipartBody(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.NormalizedMethod(), finalURL, bytes.NewReader(body))
	if err != nil {
		return nil, Wrap(KindBuildRequest, "upload_request.new_http_request", err)
	}

	applyHeaders(httpReq, req.Headers)
	if httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	return httpReq, nil
}

// BuildDownloadHTTPRequest ???????????
func BuildDownloadHTTPRequest(ctx context.Context, req DownloadRequest) (*http.Request, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	return BuildHTTPRequest(ctx, req.Request)
}

// EncodeJSONBody ???? JSON ????
func EncodeJSONBody(body any) ([]byte, string, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, "", Wrap(KindEncode, "request.encode_json", err)
	}
	return payload, ContentTypeJSON, nil
}

// EncodeFormBody ??????????
func EncodeFormBody(form map[string]string) ([]byte, string, error) {
	values := url.Values{}
	for key, value := range form {
		values.Set(key, value)
	}

	return []byte(values.Encode()), ContentTypeForm, nil
}

// EncodeMultipartBody ???? multipart ????
func EncodeMultipartBody(req UploadRequest) ([]byte, string, error) {
	if err := req.Validate(); err != nil {
		return nil, "", err
	}

	buffer := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buffer)

	for key, value := range req.FormBody {
		if err := writer.WriteField(key, value); err != nil {
			_ = writer.Close()
			return nil, "", Wrap(KindEncode, "upload_request.write_field", err)
		}
	}

	for _, file := range req.Files {
		part, err := writer.CreateFormFile(file.FieldName, file.FileName)
		if err != nil {
			_ = writer.Close()
			return nil, "", Wrap(KindEncode, "upload_request.create_form_file", err)
		}
		if _, err := io.Copy(part, file.Reader); err != nil {
			_ = writer.Close()
			return nil, "", Wrap(KindEncode, "upload_request.copy_file", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", Wrap(KindEncode, "upload_request.close_writer", err)
	}

	return buffer.Bytes(), writer.FormDataContentType(), nil
}

// encodeRequestBody ???????????????
func encodeRequestBody(req Request) (io.Reader, string, error) {
	switch {
	case req.JSONBody != nil:
		payload, contentType, err := EncodeJSONBody(req.JSONBody)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(payload), normalizeContentType(req.ContentType, contentType), nil
	case len(req.FormBody) > 0:
		payload, contentType, err := EncodeFormBody(req.FormBody)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(payload), normalizeContentType(req.ContentType, contentType), nil
	case len(req.RawBody) > 0:
		return bytes.NewReader(req.RawBody), strings.TrimSpace(req.ContentType), nil
	default:
		return nil, strings.TrimSpace(req.ContentType), nil
	}
}

// applyHeaders ????????
func applyHeaders(req *http.Request, headers map[string]string) {
	for key, value := range headers {
		req.Header.Set(key, value)
	}
}

// normalizeContentType ??????????
func normalizeContentType(current string, fallback string) string {
	current = strings.TrimSpace(current)
	if current != "" {
		return current
	}
	return fallback
}
