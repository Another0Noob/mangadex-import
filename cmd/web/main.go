package main

func main() {
	if err := startServer(); err != nil {
		panic(err)
	}
}
