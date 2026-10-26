package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	fmt.Println(runtime.NumCPU())

	fmt.Println(runtime.GOMAXPROCS(1))

	time.Sleep(3 * time.Second)

	fmt.Println(runtime.GOMAXPROCS(2))

	time.Sleep(1 * time.Second)

	go func() {
		fmt.Printf("\n\nSTARTED\n\n")
		time.Sleep(2 * time.Second)
	}()

	fmt.Println(runtime.NumGoroutine())

	fmt.Println(runtime.GOMAXPROCS(1))

	time.Sleep(2 * time.Second)
}
