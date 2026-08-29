package repository

import "errors"

// ErrNotFound is returned when a document or object does not exist.
var ErrNotFound = errors.New("not found")
