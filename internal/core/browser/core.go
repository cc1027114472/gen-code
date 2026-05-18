package browser

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Core owns real controlled browser behavior for runtime and desktop.
type Core interface {
	State(context.Context) (Snapshot, error)
	Open(context.Context, OpenRequest) (Snapshot, error)
	Navigate(context.Context, NavigateRequest) (Snapshot, error)
	Back(context.Context, TabRequest) (Snapshot, error)
	Forward(context.Context, TabRequest) (Snapshot, error)
	Reload(context.Context, TabRequest) (Snapshot, error)
	CloseTab(context.Context, TabRequest) (Snapshot, error)
	ActivateTab(context.Context, TabRequest) (Snapshot, error)
	Click(context.Context, ClickRequest) (Snapshot, error)
	Type(context.Context, TypeRequest) (Snapshot, error)
	Extract(context.Context, ExtractRequest) (ExtractResult, error)
	Screenshot(context.Context, ScreenshotRequest) (ScreenshotResult, error)
}

func normalizeURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("%w: empty url", ErrURLNotAllowed)
	}
	if strings.HasPrefix(value, "localhost:") || strings.HasPrefix(value, "127.0.0.1:") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrURLNotAllowed, err)
	}
	if !allowedURL(parsed) {
		return "", fmt.Errorf("%w: %s", ErrURLNotAllowed, value)
	}
	return parsed.String(), nil
}

func allowedURL(parsed *url.URL) bool {
	if parsed == nil {
		return false
	}
	if !strings.EqualFold(parsed.Scheme, "http") {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "127.0.0.1" || host == "localhost"
}
