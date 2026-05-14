package xhttpclient

import (
	"context"
	"net/http"

	"llmtrace/internal/platform/xlog"
)

type Client interface {
	Do(ctx context.Context, req Request) (*Response, error)
	GetJSON(ctx context.Context, req Request, out any) error
	PostJSON(ctx context.Context, req Request, out any) error
	PutJSON(ctx context.Context, req Request, out any) error
	Delete(ctx context.Context, req Request, out any) error
	PostForm(ctx context.Context, req Request, out any) error
	Upload(ctx context.Context, req UploadRequest, out any) error
	Download(ctx context.Context, req DownloadRequest) (*FileResult, error)
	Submit(ctx context.Context, req Request) (*Future, error)
	SubmitBatch(ctx context.Context, reqs []Request) ([]*Future, error)
}

// client ? HTTP ?????????
type client struct {
	httpClient *http.Client
	log        xlog.Logger
	cfg        Config
	pool       *Pool
}

// New ???? HTTP ??????
func New(httpClient *http.Client, log xlog.Logger, cfg Config) Client {
	normalized := cfg.Normalize()
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: normalized.Timeout,
		}
	}

	if httpClient.Timeout <= 0 {
		httpClient.Timeout = normalized.Timeout
	}

	c := &client{
		httpClient: httpClient,
		log:        log,
		cfg:        normalized,
	}
	c.pool = NewPool(c, normalized, log)
	return c
}
