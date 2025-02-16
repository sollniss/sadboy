package chain3

import "pkgtest/pkg2"

func Func() {
	if pkg2.B() != nil {
		panic("chain3")
	}
}
