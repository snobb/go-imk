// Package ratelimit adds a rate limiting with given RPS rate.
package ratelimit

import (
	"errors"
	"sync"
	"time"
)

var ErrRateLimitExceeded = errors.New("rate limit exceeded")

// RLimit implements a fixed window rate limiter. It's a simple implementation and is generally
// lacking precision and can lead to up to double limit per given second. But here we just need to
// make sure we don't invoke command per even but rather a comman per burst of events, so this
// implementation should be sufficient.
// Otherwise we could use a Token Bucket implementation that provides a steady rate of execution.
type RLimit struct {
	limit    int           // max leases per interval
	rps      int           // current RPS
	lastTime time.Time     // timestamp of last reset
	interval time.Duration // reset interval

	mu sync.Mutex
}

// New creates a new instance of rate limiter.
func New(limit int, interval time.Duration) *RLimit {
	return &RLimit{
		limit:    limit,
		interval: interval,
		rps:      limit,
		lastTime: time.Now(),
		mu:       sync.Mutex{},
	}
}

// Lease tries to get a lease for a request. On success nil is retured or otherwise if the limit is exceeded,
// an ErrRateLimitExceeded error is returned.
func (s *RLimit) Lease(n int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n > s.limit {
		return 0, ErrRateLimitExceeded
	}

	now := time.Now()

	// reset time frame
	if now.Sub(s.lastTime) >= s.interval {
		s.rps = s.limit
		s.lastTime = now
	}

	if s.rps < n {
		return 0, ErrRateLimitExceeded
	}

	s.rps -= n
	return n, nil
}
