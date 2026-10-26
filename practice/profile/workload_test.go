package main

import "testing"

var benchmarkResult int

func BenchmarkPrimes(b *testing.B) {
	for _, impl := range []struct {
		name string
		fn   func(int) int
	}{
		{"slow", countPrimesSlow},
		{"fast", countPrimesFast},
	} {
		b.Run(impl.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchmarkResult = impl.fn(50_000)
			}
		})
	}
}

func BenchmarkAllocations(b *testing.B) {
	for _, impl := range []struct {
		name string
		fn   func(int, int) int
	}{
		{"wasteful", allocateWastefully},
		{"reuse", allocateWithReuse},
	} {
		b.Run(impl.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchmarkResult = impl.fn(1_000, 1_024)
			}
		})
	}
}

func TestPrimeImplementationsAgree(t *testing.T) {
	for _, limit := range []int{0, 1, 2, 10, 100, 10_000} {
		want := countPrimesSlow(limit)
		if got := countPrimesFast(limit); got != want {
			t.Fatalf("limit %d: fast=%d, slow=%d", limit, got, want)
		}
	}
}
