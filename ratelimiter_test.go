package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"golang.org/x/time/rate"
)

func TestRateLimiterStopsRejectedRequest(t *testing.T) {
	var calls int32
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RateLimiterMiddleware(rate.Limit(0), 0)(next)

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, recorder.Code)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("expected the protected handler not to run, but it ran %d time(s)", got)
	}
}

func TestRateLimiterHandlesConcurrentRequests(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RateLimiterMiddleware(rate.Inf, 1)(next)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "192.0.2.1:1234"
			handler.ServeHTTP(httptest.NewRecorder(), req)
		}()
	}
	wg.Wait()
}
