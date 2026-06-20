package main

import (
	"fmt"
	"runtime"
	"time"
)

func printMem() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Alloc=%d KB TotalAlloc=%d KB NumGC=%d\n",
		m.Alloc/1024,
		m.TotalAlloc/1024,
		m.NumGC,
	)
}

func main() {
	printMem()

	for i := 0; i < 10; i++ {
		_ = make([]byte, 10*1024*1024)
		printMem()
		time.Sleep(200 * time.Millisecond)
	}

	runtime.GC()
	printMem()
}
