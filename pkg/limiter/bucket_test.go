package limiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenBucketAllowConsumesTokens(t *testing.T) {
	tb := &TokenBucket{
		capacity: 3,
		tokens:   3,
		rate:     1,
	}

	for i := 0; i < 3; i++ {
		if !tb.Allow() {
			t.Fatalf("expected Allow() to be true at i=%d", i)
		}
	}

	if tb.Allow() {
		t.Fatalf("expected Allow() to be false after tokens are exhausted")
	}
}

func TestTokenBucketRefillIsCappedByCapacity(t *testing.T) {
	tb := NewTokenBucket(1, 10)

	if !tb.Allow() {
		t.Fatalf("expected initial Allow() to be true")
	}
	if tb.Allow() {
		t.Fatalf("expected Allow() to be false after draining capacity")
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		if tb.Allow() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected token to be refilled within timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if tb.Allow() {
		t.Fatalf("expected Allow() to be false due to capacity cap")
	}
}

func TestTokenBucketConcurrentAllowDoesNotExceedCapacity(t *testing.T) {
	tb := &TokenBucket{
		capacity: 1000,
		tokens:   1000,
		rate:     0,
	}

	var okCount int64
	var wg sync.WaitGroup

	workers := 50
	perWorker := 200
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				if tb.Allow() {
					atomic.AddInt64(&okCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	if got := atomic.LoadInt64(&okCount); got != 1000 {
		t.Fatalf("expected exactly 1000 successful Allows, got %d", got)
	}
}
