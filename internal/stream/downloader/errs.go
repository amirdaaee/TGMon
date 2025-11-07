// Package downloader defines downloader-specific errors.
package downloader

import "fmt"

// ErrFloodWaitTooLong indicates the flood wait exceeded the acceptable
// threshold and the caller should try another worker or back off.
type ErrFloodWaitTooLong struct {
	expected int
	actual   float64
}

func (efw *ErrFloodWaitTooLong) Error() string {
	return fmt.Sprintf("flood wait too long: %d vs %f", efw.expected, efw.actual)
}
