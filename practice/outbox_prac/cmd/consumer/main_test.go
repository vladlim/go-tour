package main

import (
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	if got := backoff(1); got != time.Second {
		t.Fatalf("backoff(1) = %s, want 1s", got)
	}
	if got := backoff(10); got != 30*time.Second {
		t.Fatalf("backoff(10) = %s, want 30s cap", got)
	}
}
