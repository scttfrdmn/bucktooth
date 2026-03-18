package gateway

import (
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter implements per-user token-bucket rate limiting.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

// NewRateLimiter creates a per-user rate limiter.
// requestsPerMinute is the sustained request rate; burst is the maximum burst size.
func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60
	}
	if burst <= 0 {
		burst = 10
	}
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(float64(requestsPerMinute) / 60.0),
		burst:    burst,
	}
}

// Allow returns true if the user is within their rate limit.
// It lazily creates a per-user limiter on first call.
func (rl *RateLimiter) Allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	l, ok := rl.limiters[userID]
	if !ok {
		l = rate.NewLimiter(rl.rps, rl.burst)
		rl.limiters[userID] = l
	}
	return l.Allow()
}
