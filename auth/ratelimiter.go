package auth

import (
	"net/http"
	"sync"
	"time"
)

// --- Rate Limiter ---

type RateLimiter struct {
	requests map[string]int
	mutex    sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]int),
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ipAddress := r.RemoteAddr

		rl.mutex.Lock()
		count, exists := rl.requests[ipAddress]
		if !exists {
			rl.requests[ipAddress] = 1
			rl.mutex.Unlock()
			go rl.resetCount(ipAddress)
			next.ServeHTTP(w, r)
			return
		}

		if count >= 5 { // Allow 5 requests per minute
			rl.mutex.Unlock()
			RespondWithError(w, http.StatusTooManyRequests, "Too many requests")
			return
		}

		rl.requests[ipAddress] = count + 1
		rl.mutex.Unlock()

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) resetCount(ipAddress string) {
	time.Sleep(1 * time.Minute)
	rl.mutex.Lock()
	delete(rl.requests, ipAddress)
	rl.mutex.Unlock()
}
