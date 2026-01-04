// Package rlimit adds a rate limiting with given RPS.
package ratelimit

import (
	"context"
	"errors"
	"time"
)

type RLimit struct {
	limit    int
	interval time.Duration
	rps      int
	lastTime time.Time
}

func New(limit int, interval time.Duration) *RLimit {
	return &RLimit{
		limit:    limit,
		interval: interval,
		rps:      limit,
		lastTime: time.Now(),
	}
}

func (s *RLimit) Lease(ctx context.Context, n int) (int, error) {
	if n > s.limit {
		return 0, errors.New("impossible lease requested")
	}

	now := time.Now()

	// reset time frame
	if now.Sub(s.lastTime) >= s.interval {
		s.rps = s.limit
		s.lastTime = now
	}

	if s.rps < n {
		return 0, errors.New("Rate limit exceeded")
	}

	s.rps -= n
	return n, nil
}
