package pkg2 // want package:"graph" package:"types"

import (
	"pkgtest/pkg1"
	"slices"
)

func B() error {
	err := pkg1.A()
	if err != nil {
		return err
	}
	return nil
}

func Caller2(f func() error) error {
	return f()
}

func Caller(f func() error) error {
	slices.SortFunc([]string{"a"}, func(a, b string) int {
		if f() != f() {
			return 0
		}
		return 1
	})
	return nil
}

func GenericCaller[T any](f func() T) T {
	return f()
}
