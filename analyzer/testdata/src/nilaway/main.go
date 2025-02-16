package main

func main() {
	var i *int
	A(func() {
		print(*i)
	})
}

func A(f func()) {
	f()
}
