package caller

import "callers/other"

func Reflect1(s Param) Result { // want "Reflect1 does not call callee function"
	reflect1(other.CallCallee)
	return nil
}

func reflect1(f any) {
	if fn, ok := f.(func()); ok {
		fn()
	}
}
