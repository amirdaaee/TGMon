package stream

import "errors"

var (
	// all retry attempts failed.
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")

	// the pipe was closed and all data was consumed.
	ErrPipeDrained = errors.New("pipe drained")
)
