package xhttpclient

import (
	"context"
	"sync"

	"llmtrace/internal/platform/xlog"
)

// job ???????????????
type job struct {
	ctx    context.Context
	req    Request
	future *Future
}

// Pool ?????????????
type Pool struct {
	queue    chan job
	client   *client
	log      xlog.Logger
	shutdown sync.Once
}

// NewPool ??????????
func NewPool(c *client, cfg Config, log xlog.Logger) *Pool {
	p := &Pool{
		queue:  make(chan job, cfg.QueueSize),
		client: c,
		log:    log,
	}

	for i := 0; i < cfg.Workers; i++ {
		go p.worker()
	}

	return p
}

// Submit ?????????
func (p *Pool) Submit(ctx context.Context, req Request) (*Future, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	future := NewFuture()
	item := job{
		ctx:    ctx,
		req:    req,
		future: future,
	}

	select {
	case p.queue <- item:
		logAsyncSubmit(ctx, p.log, req, len(p.queue))
		return future, nil
	default:
		err := WrapQueueFull()
		logAsyncSubmitError(ctx, p.log, req, err)
		return nil, err
	}
}

// worker ?????????????
func (p *Pool) worker() {
	for item := range p.queue {
		resp, err := p.client.Do(item.ctx, item.req)
		if err == nil && resp != nil {
			logAsyncComplete(item.ctx, p.log, item.req, resp)
		}

		item.future.Resolve(Result{
			Request:  item.req,
			Response: resp,
			Error:    err,
		})
	}
}
