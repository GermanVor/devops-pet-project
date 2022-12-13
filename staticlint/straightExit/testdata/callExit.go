package main

import (
	"fmt"
	"log"
	"os"
)

type A struct {
}

func (a *A) main() {
}

func (a *A) Qwe() {
	a.main()
}

func Qwerty() {
}

func Exit() int {
	return 23
}

func main() {
	Exit()
	fmt.Println("qwe")

	Exit := func() {
	}

	log.Default()
	Exit()
	os.Exit(23)

	Exit()
}
