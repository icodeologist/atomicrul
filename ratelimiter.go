package main

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

func getIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func RateLimiterMiddleware(limit rate.Limit, burst int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	limiters := make(map[string]*rate.Limiter)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)

			mu.Lock()
			limiter, exists := limiters[ip]
			if !exists {
				limiter = rate.NewLimiter(limit, burst)
				limiters[ip] = limiter
			}
			mu.Unlock()

			if !limiter.Allow() {
				writeJson(w, http.StatusTooManyRequests, apiError{Err: "Too many requests."})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
