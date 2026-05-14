package xhttpclient

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ReadResponse ???????? HTTP ???
func ReadResponse(resp *http.Response, duration time.Duration) (*Response, error) {
	if resp == nil {
		return nil, Wrap(KindDecode, "response.read", errNilHTTPResponse)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, Wrap(KindDecode, "response.read", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    cloneHeader(resp.Header),
		Body:       body,
		Duration:   duration,
	}, nil
}

// DecodeJSONBody ???? JSON ?????
func DecodeJSONBody(body []byte, out any) error {
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return Wrap(KindDecode, "response.decode_json", err)
	}
	return nil
}

// DecodeJSONResponse ?????????? JSON ???
func DecodeJSONResponse(resp *Response, out any) error {
	if resp == nil {
		return Wrap(KindDecode, "response.decode_json", errNilResponse)
	}
	return DecodeJSONBody(resp.Body, out)
}

// SaveHTTPResponse ??? HTTP ??????????
func SaveHTTPResponse(resp *http.Response, savePath string, duration time.Duration) (*FileResult, error) {
	if resp == nil {
		return nil, Wrap(KindFileIO, "response.save_http", errNilHTTPResponse)
	}
	defer resp.Body.Close()

	if err := ensureParentDir(savePath); err != nil {
		return nil, err
	}

	file, err := os.Create(savePath)
	if err != nil {
		return nil, Wrap(KindFileIO, "response.create_file", err)
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return nil, Wrap(KindFileIO, "response.copy_to_file", err)
	}

	return &FileResult{
		SavePath:   savePath,
		Size:       size,
		StatusCode: resp.StatusCode,
		Duration:   duration,
	}, nil
}

// SaveResponseBody ????????????
func SaveResponseBody(resp *Response, savePath string) (*FileResult, error) {
	if resp == nil {
		return nil, Wrap(KindFileIO, "response.save_body", errNilResponse)
	}
	if err := ensureParentDir(savePath); err != nil {
		return nil, err
	}
	if err := os.WriteFile(savePath, resp.Body, 0o644); err != nil {
		return nil, Wrap(KindFileIO, "response.write_file", err)
	}

	return &FileResult{
		SavePath:   savePath,
		Size:       int64(len(resp.Body)),
		StatusCode: resp.StatusCode,
		Duration:   resp.Duration,
	}, nil
}

// ensureParentDir ???????????
func ensureParentDir(savePath string) error {
	if savePath == "" {
		return Wrap(KindFileIO, "response.ensure_parent_dir", errMissingSavePath)
	}

	dir := filepath.Dir(savePath)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Wrap(KindFileIO, "response.mkdir_all", err)
	}

	return nil
}

// cloneHeader ???? HTTP ????
func cloneHeader(header http.Header) map[string][]string {
	if len(header) == 0 {
		return nil
	}

	cloned := make(map[string][]string, len(header))
	for key, values := range header {
		cloned[key] = append([]string(nil), values...)
	}

	return cloned
}
