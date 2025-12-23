// Package rlimit adds a rate limiting with given RPS.
package ratelimit

import (
	"context"
	"errors"
	"time"
)

type RLimit struct {
	limit    int
	rps      int
	lastTime int64
}

func New(limit int) *RLimit {
	return &RLimit{
		limit:    limit,
		rps:      limit,
		lastTime: time.Now().Unix(),
	}
}

func (s *RLimit) Lease(ctx context.Context, n int) (int, error) {
	if n > s.limit {
		return 0, errors.New("impossible lease requested")
	}

	now := time.Now().Unix()

	// reset time frame
	if now != s.lastTime {
		s.rps = s.limit
		s.lastTime = now
	}

	if s.rps < n {
		return 0, errors.New("Rate limit exceeded")
	}

	s.rps -= n
	return n, nil
}
