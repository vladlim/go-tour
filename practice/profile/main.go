package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

var result int

func main() {
	mode := flag.String("mode", "cli", "cli or server")
	addr := flag.String("addr", "localhost:6060", "diagnostic HTTP address")
	impl := flag.String("impl", "slow", "slow or fast")
	task := flag.String("task", "cpu", "cpu or alloc")
	n := flag.Int("n", 100_000, "prime limit, or allocation item count")
	repeat := flag.Int("repeat", 100, "number of workload repetitions")
	cpuProfile := flag.String("cpuprofile", "", "write CPU profile to this file")
	memProfile := flag.String("memprofile", "", "write heap profile to this file after the work")
	flag.Parse()

	work := func() int { return runWork(*task, *impl, *n, *repeat) }
	if *mode == "server" {
		runServer(*addr, work)
		return
	}
	runCLI(work, *cpuProfile, *memProfile)
}

func runWork(task, impl string, n, repeat int) int {
	total := 0
	for i := 0; i < repeat; i++ {
		switch {
		case task == "cpu" && impl == "slow":
			total += countPrimesSlow(n)
		case task == "cpu" && impl == "fast":
			total += countPrimesFast(n)
		case task == "alloc" && impl == "slow":
			total += allocateWastefully(n, 1_024)
		case task == "alloc" && impl == "fast":
			total += allocateWithReuse(n, 1_024)
		default:
			log.Fatalf("unsupported -task=%q -impl=%q", task, impl)
		}
	}
	return total
}

func runCLI(work func() int, cpuPath, memPath string) {
	if cpuPath != "" {
		file := create(cpuPath)
		if err := pprof.StartCPUProfile(file); err != nil {
			log.Fatal(err)
		}
		defer func() {
			pprof.StopCPUProfile()
			if err := file.Close(); err != nil {
				log.Fatal(err)
			}
		}()
	}

	started := time.Now()
	result = work()
	fmt.Printf("result=%d duration=%s\n", result, time.Since(started))

	if memPath != "" {
		runtime.GC() // make this example's live-heap snapshot easier to interpret
		file := create(memPath)
		if err := pprof.WriteHeapProfile(file); err != nil {
			log.Fatal(err)
		}
		if err := file.Close(); err != nil {
			log.Fatal(err)
		}
	}
}

func runServer(addr string, work func() int) {
	http.HandleFunc("/work", func(w http.ResponseWriter, _ *http.Request) {
		started := time.Now()
		value := work()
		fmt.Fprintf(w, "result=%d duration=%s\n", value, time.Since(started))
	})
	log.Printf("workload: http://%s/work", addr)
	log.Printf("profiles: http://%s/debug/pprof/", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func create(path string) *os.File {
	file, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	return file
}
