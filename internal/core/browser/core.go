package browser

import (
	"context"
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
