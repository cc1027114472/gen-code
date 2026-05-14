package xhttpclient

import "context"

// Submit ?????????
func (c *client) Submit(ctx context.Context, req Request) (*Future, error) {
	if c == nil || c.pool == nil {
		return nil, NewError(KindValidation, "client.submit", "async pool is not initialized")
	}
	return c.pool.Submit(ctx, req)
}

// SubmitBatch ???????????
func (c *client) SubmitBatch(ctx context.Context, reqs []Request) ([]*Future, error) {
	futures := make([]*Future, 0, len(reqs))
	for _, req := range reqs {
		future, err := c.Submit(ctx, req)
		if err != nil {
			return futures, err
		}
		futures = append(futures, future)
	}
	return futures, nil
}
