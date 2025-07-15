package web

import (
	"fmt"
)

type HttpErr struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"msg"`
}

func (e HttpErr) Error() string {
	return e.Message
}

func NewHttpError(err error, statusCode int) HttpErr {
	e := HttpErr{
		StatusCode: statusCode,
		Message:    err.Error(),
	}
	return e
}

var ErrNotImplemented = fmt.Errorf("not supported method")
