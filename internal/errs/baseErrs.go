package errs

import "reflect"

type IErr interface {
	error
}

func IsErr(val error, target IErr) bool {
	return reflect.TypeOf(val) == reflect.TypeOf(target)
}
