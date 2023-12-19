package constants

import "errors"

var ErrNil = errors.New("result is nil")

func IsErrNil(err error) bool {
	switch {
	case
		errors.Is(err, ErrNil):
		return true
	default:
		return false
	}
}
