package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"go-imk/internal/ratelimit"
	"go-imk/test/assert"
)

func TestStore_Lease(t *testing.T) {
	tests := []struct {
		name    string
		limit   int
		n       int
		nrun    int
		wantErr bool
	}{
		{
			name:    "should return error on exceeded rps",
			limit:   1,
			n:       1,
			nrun:    6,
			wantErr: true,
		},
		{
			name:    "should run without errors",
			limit:   6,
			n:       2,
			nrun:    3,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := ratelimit.New(tt.limit, time.Second)
			for i := 0; i < tt.nrun; i++ {
				n, err := rl.Lease(context.Background(), tt.n)

				if i >= tt.limit && tt.wantErr {
					assert.Error(t, err)
					return
				}

				assert.Equal(t, n, tt.n)
				assert.NoError(t, err)
			}
		})
	}
}
