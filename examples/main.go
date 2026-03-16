package main

import (
	"fmt"

	"github.com/oarkflow/naturaldate"
)

func main() {
	fmt.Println(naturaldate.Parse("yesterday", naturaldate.Options{}))
}
