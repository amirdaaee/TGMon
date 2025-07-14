package downloader

import "fmt"

type ErrFloodWaitTooLong struct {
	expected int
	actual   float64
}

func (efw *ErrFloodWaitTooLong) Error() string {
	return fmt.Sprintf("flood wait too long: %d vs %f", efw.expected, efw.actual)
}
