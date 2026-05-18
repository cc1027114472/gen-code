package browsercore

import core "llmtrace/internal/core/browser"

type (
	Snapshot          = core.Snapshot
	Tab               = core.Tab
	OpenRequest       = core.OpenRequest
	NavigateRequest   = core.NavigateRequest
	TabRequest        = core.TabRequest
	ClickRequest      = core.ClickRequest
	TypeRequest       = core.TypeRequest
	ExtractRequest    = core.ExtractRequest
	ScreenshotRequest = core.ScreenshotRequest
	ExtractResult     = core.ExtractResult
	ScreenshotResult  = core.ScreenshotResult
	Core              = core.Core
)

var (
	ErrTabNotFound            = core.ErrTabNotFound
	ErrURLNotAllowed          = core.ErrURLNotAllowed
	ErrSelectorNotFound       = core.ErrSelectorNotFound
	ErrElementNotInteractable = core.ErrElementNotInteractable
	ErrScreenshotFailed       = core.ErrScreenshotFailed
	ErrExtractionFailed       = core.ErrExtractionFailed
	ErrSessionUnavailable     = core.ErrSessionUnavailable
)

func NewDriver() Core {
	return core.NewDriver()
}
