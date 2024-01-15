package main

import (
	"fmt"
	"strings"
)

func main() {
	str := "hello world"

	fmt.Println(strings.TrimRight(str, "rld"))
}
