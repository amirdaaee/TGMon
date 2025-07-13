package stream

import "fmt"

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

func NewStreamError(msg string, err error) *StreamError {
	return &StreamError{Msg: msg, Err: err}
}

var ErrNoThumbnail = fmt.Errorf("doc doesnt have any thumbnail")
