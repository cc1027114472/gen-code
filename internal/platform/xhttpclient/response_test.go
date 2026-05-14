package xhttpclient

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestFutureWaitReturnsResolvedResult ????????????
func TestFutureWaitReturnsResolvedResult(t *testing.T) {
	future := NewFuture()
	want := Result{
		Request: Request{Name: "demo"},
		Error:   errors.New("boom"),
	}

	go future.Resolve(want)

	got := future.Wait()

	require.Equal(t, want.Request.Name, got.Request.Name)
	require.EqualError(t, got.Error, "boom")
}

// TestFutureWaitTimeoutReturnsFalseWhenPending ????????????
func TestFutureWaitTimeoutReturnsFalseWhenPending(t *testing.T) {
	future := NewFuture()

	got, ok := future.WaitTimeout(20 * time.Millisecond)

	require.False(t, ok)
	require.Equal(t, Result{}, got)
}

// TestFutureWaitTimeoutReturnsResolvedResult ????????????
func TestFutureWaitTimeoutReturnsResolvedResult(t *testing.T) {
	future := NewFuture()
	want := Result{
		Response: &Response{StatusCode: 200},
	}
	future.Resolve(want)

	got, ok := future.WaitTimeout(time.Second)

	require.True(t, ok)
	require.Equal(t, 200, got.Response.StatusCode)
}
