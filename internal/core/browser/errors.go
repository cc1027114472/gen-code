package browser

import "errors"

var (
	ErrTabNotFound            = errors.New("browser: tab not found")
	ErrURLNotAllowed          = errors.New("browser: url not allowed")
	ErrSelectorNotFound       = errors.New("browser: selector not found")
	ErrElementNotInteractable = errors.New("browser: element not interactable")
	ErrScreenshotFailed       = errors.New("browser: screenshot failed")
	ErrExtractionFailed       = errors.New("browser: extraction failed")
	ErrSessionUnavailable     = errors.New("browser: session unavailable")
)
