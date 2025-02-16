package pkg3 // want package:"graph" package:"types"

import (
	"pkgtest/pkg2"
)

type Return error

//func C() error {
//	return pkg2.B()
//}

//func C() error {
//	slices.SortFunc([]string{"a"}, func(a, b string) int {
//		if pkg2.B() != pkg2.B() {
//			return 0
//		}
//		return 1
//	})
//	return nil
//}

//func C() Return {
//	return caller(func() error {
//		return pkg2.B()
//	})
//}
//
//func caller(f func() error) error {
//	return f()
//}

// -------------------------------------------------

/*
	func C() Return {
		return pkg2.Caller(func() error {
			return pkg2.B()
		})
	}

	func D() Return {
		f := func() error {
			return pkg2.B()
		}
		return pkg2.Caller(f)
	}

	func E() Return {
		f := func() error {
			return pkg2.B()
		}
		return f()
	}

	func F() Return {
		return nil
	}
*/

/*
	func G() Return {
		slices.SortFunc([]string{"a"}, func(a, b string) int {
			if pkg2.B() != pkg2.B() {
				return 0
			}
			return 1
		})
		return nil
	}

	func H_1() Return {
		return pkg2.Caller(pkg2.B)
	}

	func H_2() Return {
		return pkg2.Caller(func() error {
			return pkg2.B()
		})
	}

	func H_3() Return {
		return pkg2.Caller(func() error {
			return nil
		})
	}

	func I() Return {
		chain0.Func()
		return nil
	}

	func J() Return {
		return pkg1.A()
	}

	func K() Return {
		return nil
	}
*/
func Test8() Return {
	test8(pkg2.B)
	return nil
}

func test8(f any) {
	if fn, ok := f.(func()); ok {
		fn()
	}
}
