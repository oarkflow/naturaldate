package main

import (
	"fmt"
	"time"

	"github.com/oarkflow/naturaldate"
)

func main() {
	ref := time.Date(2026, time.March, 16, 15, 4, 5, 0, time.UTC)
	result, ok := naturaldate.Parse("next friday at 5pm", naturaldate.Options{Reference: ref})
	if !ok {
		panic("parse failed")
	}
	fmt.Println(result.Time)
}
