package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("good morning")
		time.Sleep(10 * time.Second)
	}
}
