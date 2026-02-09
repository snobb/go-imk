package ratelimit_test

import (
	"testing"
	"time"

	"github.com/snobb/go-imk/internal/ratelimit"
	"github.com/snobb/go-imk/test/assert"
)

func TestTokenBucket_Lease(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		rate     int
		n        int
		nrun     int
		interval time.Duration
		delay    time.Duration
		wantErr  bool
	}{
		{
			name:     "should return error on exceeded rps",
			capacity: 2,
			rate:     2,
			n:        1,
			nrun:     3,
			interval: time.Second,
			wantErr:  true,
		},
		{
			name:     "should run without errors (with refills)",
			capacity: 2,
			rate:     4, // 4 tokens per interval.
			n:        2,
			nrun:     2,
			delay:    100 * time.Millisecond, // should refill half a bucket
			interval: 200 * time.Millisecond, // given the interval is double the delay.
			wantErr:  false,
		},
		{
			name:     "should run without errors",
			capacity: 4,
			rate:     4,
			n:        1,
			nrun:     2,
			interval: time.Second,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := ratelimit.NewTokenBucket(tt.capacity, tt.rate, 200*time.Millisecond)
			for i := 0; i < tt.nrun; i++ {
				err := tb.Lease(tt.n)

				if i >= tt.capacity && tt.wantErr {
					assert.Error(t, err)
					return
				}

				assert.NoError(t, err)

				time.Sleep(tt.delay)
			}
		})
	}
}
