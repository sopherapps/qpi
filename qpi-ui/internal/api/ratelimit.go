package api

import (
	"sync"
	"time"
)

// rateLimiter is a minimal token-bucket limiter guarding one driver's inbound
// event stream (RFC 0001 §7, Phase 5). It refills at a fixed rate up to a
// burst capacity and reports whether the next event may be processed now.
//
// One limiter belongs to one driver's listener goroutine, so it needs no
// cross-driver coordination; the mutex only keeps Allow safe if a driver ever
// grows more than one inbound reader. A non-positive rate means "unlimited",
// which lets operators disable the limit via config.
type rateLimiter struct {
	mu        sync.Mutex
	unlimited bool
	rate      float64
	burst     float64
	tokens    float64
	last      time.Time
}

// newRateLimiter builds a limiter allowing perSecond events per second with a
// bucket that holds one second's worth of burst. A perSecond <= 0 disables the
// limit.
func newRateLimiter(perSecond int) *rateLimiter {
	if perSecond <= 0 {
		return &rateLimiter{unlimited: true}
	}
	burst := float64(perSecond)
	return &rateLimiter{
		rate:   float64(perSecond),
		burst:  burst,
		tokens: burst,
		// last is zero-valued; set on the first allowAt call so the bucket
		// tracks the caller's clock, not the wall clock at construction time.
	}
}

// Allow reports whether an event may be processed now, consuming a token if so.
func (r *rateLimiter) Allow() bool {
	return r.allowAt(time.Now())
}

// allowAt is Allow with an injectable clock so the refill behaviour is
// deterministically testable.
func (r *rateLimiter) allowAt(now time.Time) bool {
	if r.unlimited {
		return true
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.last.IsZero() {
		// First call: anchor the bucket to the caller's clock.
		r.last = now
	} else if elapsed := now.Sub(r.last).Seconds(); elapsed > 0 {
		r.tokens += elapsed * r.rate
		if r.tokens > r.burst {
			r.tokens = r.burst
		}
		r.last = now
	}

	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	return false
}
