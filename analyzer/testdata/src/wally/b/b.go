package b // want package:"graph"

func B() error {
	return nil
}

func Caller(f func() error) error {
	return f()
}
