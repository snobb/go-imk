package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket implements token bucket algorithm for rate limiting.
type TokenBucket struct {
	capacity   int           // size of the bucket.
	rate       int           // refill rate.
	tokens     int           // current tokens in the bucket.
	refilledAt time.Time     // timestamp of last refill.
	interval   time.Duration // refill interval (add 'rate' tokens every 'interval').

	mu sync.Mutex
}

// NewTokenBucket returns a new instance of TokenBucket rate limiter.
func NewTokenBucket(capacity, rate int, interval time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		rate:       rate,
		tokens:     capacity,
		refilledAt: time.Now(),
		interval:   interval,
	}
}

// Lease return nil if there are enough tokens available in the buckets. Otherwise it returns
// ErrRateLimitExceeded error.
func (tb *TokenBucket) Lease(tokens int) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= tokens {
		tb.tokens -= tokens
		return nil
	}

	return ErrRateLimitExceeded
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.refilledAt)
	tokensToRefill := int((elapsed.Seconds() / tb.interval.Seconds()) * float64(tb.rate))

	if tokensToRefill > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+tokensToRefill)
		tb.refilledAt = now
	}
}
