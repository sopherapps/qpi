package api

import (
	"testing"
	"time"
)

// TestRateLimiter_EnforcesBurstThenRefills proves a limiter admits up to its
// burst immediately, rejects once the bucket is empty, and admits again after
// enough time to refill a token.
func TestRateLimiter_EnforcesBurstThenRefills(t *testing.T) {
	limiter := newRateLimiter(2)
	start := time.Unix(0, 0)

	if !limiter.allowAt(start) {
		t.Fatal("expected first event within burst to be allowed")
	}
	if !limiter.allowAt(start) {
		t.Fatal("expected second event within burst to be allowed")
	}
	if limiter.allowAt(start) {
		t.Fatal("expected third event to be dropped once the bucket is empty")
	}

	// At 2 events/sec, half a second refills exactly one token.
	if !limiter.allowAt(start.Add(500 * time.Millisecond)) {
		t.Fatal("expected an event to be allowed after the bucket refills")
	}
	if limiter.allowAt(start.Add(500 * time.Millisecond)) {
		t.Fatal("expected the very next event to be dropped again")
	}
}

// TestRateLimiter_Unlimited proves a non-positive rate disables the limit.
func TestRateLimiter_Unlimited(t *testing.T) {
	limiter := newRateLimiter(0)
	now := time.Unix(0, 0)
	for i := 0; i < 1000; i++ {
		if !limiter.allowAt(now) {
			t.Fatalf("expected unlimited limiter to always allow, dropped at %d", i)
		}
	}
}

// TestRateLimiter_CapsAtBurst proves accumulated tokens never exceed the burst
// capacity, so a long idle period cannot grant an unbounded backlog.
func TestRateLimiter_CapsAtBurst(t *testing.T) {
	limiter := newRateLimiter(5)
	start := time.Unix(0, 0)

	// Idle for a long time, then confirm only `burst` events pass at once.
	late := start.Add(time.Hour)
	allowed := 0
	for i := 0; i < 100; i++ {
		if limiter.allowAt(late) {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("expected exactly burst=5 events after long idle, got %d", allowed)
	}
}
