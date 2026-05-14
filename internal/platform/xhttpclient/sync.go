package xhttpclient

import (
	"context"
	"net/http"
	"time"
)

// Do ?????? HTTP ???
func (c *client) Do(ctx context.Context, req Request) (*Response, error) {
	httpReq, err := BuildHTTPRequest(ctx, req)
	if err != nil {
		return nil, WrapBuildError(err)
	}

	finalURL := httpReq.URL.String()
	bodyPreview := previewRequestBody(req)
	logRequest(ctx, c.log, c.cfg, req, finalURL, maskHeaders(req.HeaderClone()), bodyPreview)

	startedAt := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logError(ctx, c.log, c.cfg, req, finalURL, 0, err, bodyPreview, "")
		return nil, WrapExecuteError(err)
	}

	resp, err := ReadResponse(httpResp, time.Since(startedAt))
	if err != nil {
		logError(ctx, c.log, c.cfg, req, finalURL, httpResp.StatusCode, err, bodyPreview, "")
		return nil, WrapDecodeError(err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		statusErr := WrapStatus("request.do", resp.StatusCode, resp.Body)
		logError(ctx, c.log, c.cfg, req, finalURL, resp.StatusCode, statusErr, bodyPreview, string(resp.Body))
		return nil, statusErr
	}

	logResult(ctx, c.log, c.cfg, req, finalURL, resp)
	return resp, nil
}

// GetJSON ???? GET ????? JSON ???
func (c *client) GetJSON(ctx context.Context, req Request, out any) error {
	req.Method = http.MethodGet
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	return DecodeJSONResponse(resp, out)
}

// PostJSON ???? POST JSON ????????
func (c *client) PostJSON(ctx context.Context, req Request, out any) error {
	req.Method = http.MethodPost
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	return DecodeJSONResponse(resp, out)
}

// PutJSON ???? PUT JSON ????????
func (c *client) PutJSON(ctx context.Context, req Request, out any) error {
	req.Method = http.MethodPut
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	return DecodeJSONResponse(resp, out)
}

// Delete ???? DELETE ????????
func (c *client) Delete(ctx context.Context, req Request, out any) error {
	req.Method = http.MethodDelete
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	return DecodeJSONResponse(resp, out)
}

// PostForm ??????????????
func (c *client) PostForm(ctx context.Context, req Request, out any) error {
	req.Method = http.MethodPost
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	return DecodeJSONResponse(resp, out)
}

// Upload ???????????
func (c *client) Upload(ctx context.Context, req UploadRequest, out any) error {
	httpReq, err := BuildUploadHTTPRequest(ctx, req)
	if err != nil {
		return WrapBuildError(err)
	}

	finalURL := httpReq.URL.String()
	logRequest(ctx, c.log, c.cfg, req.Request, finalURL, maskHeaders(req.HeaderClone()), "[multipart upload]")

	startedAt := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logError(ctx, c.log, c.cfg, req.Request, finalURL, 0, err, "[multipart upload]", "")
		return WrapExecuteError(err)
	}

	resp, err := ReadResponse(httpResp, time.Since(startedAt))
	if err != nil {
		logError(ctx, c.log, c.cfg, req.Request, finalURL, httpResp.StatusCode, err, "[multipart upload]", "")
		return WrapDecodeError(err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		statusErr := WrapStatus("request.upload", resp.StatusCode, resp.Body)
		logError(ctx, c.log, c.cfg, req.Request, finalURL, resp.StatusCode, statusErr, "[multipart upload]", string(resp.Body))
		return statusErr
	}

	logResult(ctx, c.log, c.cfg, req.Request, finalURL, resp)
	return DecodeJSONResponse(resp, out)
}

// Download ???????????
func (c *client) Download(ctx context.Context, req DownloadRequest) (*FileResult, error) {
	httpReq, err := BuildDownloadHTTPRequest(ctx, req)
	if err != nil {
		return nil, WrapBuildError(err)
	}

	finalURL := httpReq.URL.String()
	bodyPreview := previewRequestBody(req.Request)
	logRequest(ctx, c.log, c.cfg, req.Request, finalURL, maskHeaders(req.HeaderClone()), bodyPreview)

	startedAt := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logError(ctx, c.log, c.cfg, req.Request, finalURL, 0, err, bodyPreview, "")
		return nil, WrapExecuteError(err)
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		resp, readErr := ReadResponse(httpResp, time.Since(startedAt))
		if readErr != nil {
			logError(ctx, c.log, c.cfg, req.Request, finalURL, httpResp.StatusCode, readErr, bodyPreview, "")
			return nil, WrapDecodeError(readErr)
		}

		statusErr := WrapStatus("request.download", resp.StatusCode, resp.Body)
		logError(ctx, c.log, c.cfg, req.Request, finalURL, resp.StatusCode, statusErr, bodyPreview, string(resp.Body))
		return nil, statusErr
	}

	result, err := SaveHTTPResponse(httpResp, req.SavePath, time.Since(startedAt))
	if err != nil {
		logError(ctx, c.log, c.cfg, req.Request, finalURL, httpResp.StatusCode, err, bodyPreview, "")
		return nil, WrapFileError(err)
	}

	logResult(ctx, c.log, c.cfg, req.Request, finalURL, &Response{
		StatusCode: result.StatusCode,
		Duration:   result.Duration,
		Body:       []byte("[download saved]"),
	})
	return result, nil
}
