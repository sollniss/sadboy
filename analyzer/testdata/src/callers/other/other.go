package other // want package:"types"

import "callers/callee"

func CallCallee() {
	callee.Callee()
}

func DoNotCallCallee() {
	callee.NotCallee()
}
