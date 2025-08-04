package main

import (
	"log"
	"net"
	"net/http"

	"golang.org/x/time/rate"
)

func getIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("Could not fetch the ip : %v\n", err.Error())
	}
	return host
}

var ipLimiterMap = make(map[string]*rate.Limiter)

func RateLimiterMiddleware(limit rate.Limit, burst int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// fetch ip
			ip := getIP(r)
			limiter, exists := ipLimiterMap[ip]
			if !exists {
				// create
				limiter = rate.NewLimiter(limit, burst)
				ipLimiterMap[ip] = limiter
			}
			if !limiter.Allow() {
				writeJson(w, 429, apiError{Err: "Too many requests."})
			}
			next.ServeHTTP(w, r)
		})
	}
}
