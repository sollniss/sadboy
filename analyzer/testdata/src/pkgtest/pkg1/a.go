package pkg1 // want package:"graph" package:"types"

func A() error {
	err := a()
	if err != nil {
		return err
	}
	return nil
}

func a() error {
	return nil
}
