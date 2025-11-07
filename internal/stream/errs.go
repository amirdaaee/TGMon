// Package stream defines error types used by the streaming subsystem.
package stream

import "fmt"

// StreamError wraps a message and an underlying error to provide context
// while preserving unwrap semantics.
type StreamError struct {
	Msg string
	Err error
}

func (e *StreamError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *StreamError) Unwrap() error {
	return e.Err
}

// NewStreamError constructs a StreamError with the provided message and cause.
func NewStreamError(msg string, err error) *StreamError {
	return &StreamError{Msg: msg, Err: err}
}

// ErrNoThumbnail indicates that the target document has no thumbnail sizes.
var ErrNoThumbnail = fmt.Errorf("doc doesnt have any thumbnail")
