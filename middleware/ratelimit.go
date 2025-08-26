package middleware

import (
	"net"
	"net/http"
	"story_ai/metrics"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	requests    map[string][]time.Time
	maxRequests int
	window      time.Duration
	mu          sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: maxRequests,
		window:      window,
	}

	// Clean up old entries periodically
	go rl.cleanup()

	return rl
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	requests := rl.requests[ip]

	// Remove old requests outside the window
	var validRequests []time.Time
	for _, reqTime := range requests {
		if now.Sub(reqTime) < rl.window {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if under the limit
	if len(validRequests) < rl.maxRequests {
		validRequests = append(validRequests, now)
		rl.requests[ip] = validRequests
		// Record allowed request
		metrics.RecordRateLimit(ip, false)
		return true
	}

	// Record blocked request
	metrics.RecordRateLimit(ip, true)
	return false
}

// cleanup removes old entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, requests := range rl.requests {
			var validRequests []time.Time
			for _, reqTime := range requests {
				if now.Sub(reqTime) < rl.window {
					validRequests = append(validRequests, reqTime)
				}
			}
			if len(validRequests) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = validRequests
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware returns a middleware function that rate limits requests
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			if !rl.Allow(ip) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusTooManyRequests)
				errorHTML := `
				<div id="story-container">
					<h1>⏰ Please Slow Down!</h1>
					<div style="background-color: #252526; padding: 20px; border-radius: 8px; margin: 20px 0;">
						<p>🤖 You're sending requests too quickly to the story generator.</p>
						<p>💡 <strong>Suggestion:</strong> Please wait a moment before trying again.</p>
						<p>⏱️ <strong>Wait time:</strong> About 60 seconds</p>
						<p>🔄 <em>This helps ensure everyone gets a chance to enjoy the stories!</em></p>
					</div>
					<button onclick="setTimeout(() => window.location.reload(), 60000)">Try Again in 1 Minute</button>
				</div>`
				w.Write([]byte(errorHTML))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// Take the first IP if there are multiple
		ip := strings.Split(xForwardedFor, ",")[0]
		ip = strings.TrimSpace(ip)
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Check X-Real-IP header
	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" && net.ParseIP(xRealIP) != nil {
		return xRealIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
