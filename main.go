package main

import (
	"fmt"
	"io"
)

type myIO struct {
	curr int
	last int
}

func (r *myIO) Read(b []byte) (int, error) {
	fmt.Printf("reader: curr=%d last=%d\n", r.curr, r.last)
	if r.curr > r.last {
		return 0, io.EOF
	}
	t := fmt.Sprintf("im: %d", r.curr)
	l := copy(b, []byte(t))
	r.curr++
	return l, nil
}
func (r *myIO) Write(p []byte) (int, error) {
	fmt.Printf("writer: %s\n", p)
	return len(p), nil
}

func main() {
	mr := myIO{0, 10}
	mw := myIO{0, 10}
	io.Copy(&mr, &mw)
}
