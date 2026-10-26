package main

// countPrimesSlow deliberately uses trial division. It is useful as a visible
// CPU hot spot, not as production-quality prime-number code.
func countPrimesSlow(limit int) int {
	count := 0
	for candidate := 2; candidate <= limit; candidate++ {
		prime := true
		for divisor := 2; divisor*divisor <= candidate; divisor++ {
			if candidate%divisor == 0 {
				prime = false
				break
			}
			if prime {
				count++
			}
		}
	}
	return count
}

// countPrimesFast uses the Sieve of Eratosthenes. It exchanges a small memory
// allocation for substantially less CPU work.
func countPrimesFast(limit int) int {
	if limit < 2 {
		return 0
	}
	composite := make([]bool, limit+1)
	for n := 2; n*n <= limit; n++ {
		if composite[n] {
			continue
		}
		for multiple := n * n; multiple <= limit; multiple += n {
			composite[multiple] = true
		}
	}
	count := 0
	for n := 2; n <= limit; n++ {
		if !composite[n] {
			count++
		}
	}
	return count
}

func allocateWastefully(items, size int) int {
	total := 0
	for i := 0; i < items; i++ {
		buf := make([]byte, size)
		for j := range buf {
			buf[j] = byte(i + j)
		}
		total += int(buf[i%size])
	}
	return total
}

func allocateWithReuse(items, size int) int {
	buf := make([]byte, size)
	total := 0
	for i := 0; i < items; i++ {
		for j := range buf {
			buf[j] = byte(i + j)
		}
		total += int(buf[i%size])
	}
	return total
}
