package caller // want package:"types"

import (
	"callers/other"
	"slices"
)

type Param string
type Result error

func Test1(s Param) Result { // OK: calls Callee directly (exported)
	other.CallCallee()
	return nil
}

func Test1_fail(s Param) Result { // want "Test1_fail does not call callee function"
	other.DoNotCallCallee()
	return nil
}

func Test2() Result { // OK: (helper) does not match Caller
	return test2("")
}

func Test2_Fail() Result { // OK: (helper) does not match Caller
	return test2_fail("")
}

func test2(_ Param) Result { // OK: calls Calee directly (unexported)
	other.CallCallee()
	return nil
}

func test2_fail(_ Param) Result { // want "test2_fail does not call callee function"
	other.DoNotCallCallee()
	return nil
}

func Test3(s Param) Result { // OK: calls anonymous function that calls Callee
	f := func() {
		other.CallCallee()
	}
	f()
	return nil
}

func Test3_fail(s Param) Result { // want "Test3_fail does not call callee function"
	f := func() {
		other.DoNotCallCallee()
	}
	f()
	return nil
}

func Test4(s Param) Result { // OK: calls Callee in defer
	defer other.CallCallee()
	return nil
}

func Test4_fail(s Param) Result { // want "Test4_fail does not call callee function"
	defer other.DoNotCallCallee()
	return nil
}

func Test5(s Param) Result { // OK: calls Callee in anonymous function passed to another (generic) function
	slices.SortFunc([]string{"a"}, func(a, b string) int {
		other.CallCallee()
		return 1
	})
	return nil
}

func Test5_fail(s Param) Result { // want "Test5_fail does not call callee function"
	slices.SortFunc([]string{"a"}, func(a, b string) int {
		other.DoNotCallCallee()
		return 1
	})
	return nil
}

type Dummy struct{}

func (d Dummy) Test6(s Param) Result { // OK: calls Callee (method, struct receiver)

	other.CallCallee()
	return nil
}

func (d Dummy) Test6_fail(s Param) Result { // want "Test6_fail does not call callee function"

	other.DoNotCallCallee()
	return nil
}

func (d *Dummy) Test7(s Param) Result { // OK: calls Callee (method, pointer receiver)

	other.CallCallee()
	return nil
}

func (d *Dummy) Test7_fail(s Param) Result { // want "Test7_fail does not call callee function"

	other.DoNotCallCallee()
	return nil
}

func Test8(s Param) Result { // OK: calls Callee (function)
	d := &Dummy{}
	test8(d)
	return nil
}

func (d *Dummy) test8() { // want "Test7_fail does not call callee function"
	other.CallCallee()
}

type itest8 interface {
	test8()
}

func test8(i itest8) { // OK: calls Callee (interface)
	i.test8()
}

func Test8_fail(s Param) Result { // want "Test8_fail does not call callee function"

	d := &Dummy{}
	test8_fail(d)
	return nil
}

type itest8_fail interface {
	test8_fail()
}

func (d *Dummy) test8_fail() {
	other.DoNotCallCallee()
}

func test8_fail(i itest8_fail) {
	i.test8_fail()
}
