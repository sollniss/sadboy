package a // want package:"graph"

import (
	"wally/b"
)

type Return error

func A() Return {
	return b.Caller(func() error {
		return b.B()
	})
}
