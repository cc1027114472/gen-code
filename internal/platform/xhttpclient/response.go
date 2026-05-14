package xhttpclient

import (
	"sync"
	"time"
)

// Response ??????? HTTP ???
type Response struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
	Duration   time.Duration
}

// FileResult ?????????????
type FileResult struct {
	SavePath   string
	Size       int64
	StatusCode int
	Duration   time.Duration
}

// Result ??????????????
type Result struct {
	Request  Request
	Response *Response
	Error    error
}

// Future ?????????????
type Future struct {
	done   chan struct{}
	once   sync.Once
	result Result
	mu     sync.RWMutex
}

// NewFuture ???????????
func NewFuture() *Future {
	return &Future{
		done: make(chan struct{}),
	}
}

// Resolve ???????????
func (f *Future) Resolve(result Result) {
	if f == nil {
		return
	}

	f.once.Do(func() {
		f.mu.Lock()
		f.result = result
		f.mu.Unlock()
		close(f.done)
	})
}

// Wait ?????????????
func (f *Future) Wait() Result {
	if f == nil {
		return Result{}
	}

	<-f.done
	return f.snapshot()
}

// WaitTimeout ???????????????
func (f *Future) WaitTimeout(timeout time.Duration) (Result, bool) {
	if f == nil {
		return Result{}, false
	}

	if timeout <= 0 {
		select {
		case <-f.done:
			return f.snapshot(), true
		default:
			return Result{}, false
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-f.done:
		return f.snapshot(), true
	case <-timer.C:
		return Result{}, false
	}
}

// Done ?????????????
func (f *Future) Done() <-chan struct{} {
	if f == nil {
		return nil
	}
	return f.done
}

// snapshot ?????????????
func (f *Future) snapshot() Result {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.result
}

// HeaderClone ???????????????
func (r *Response) HeaderClone() map[string][]string {
	if r == nil || len(r.Headers) == 0 {
		return nil
	}

	cloned := make(map[string][]string, len(r.Headers))
	for key, values := range r.Headers {
		cloned[key] = append([]string(nil), values...)
	}

	return cloned
}
