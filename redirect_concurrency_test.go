package main

import (
	"net/http"
	"sync"
	"testing"
)

func TestRedirectConcurrentClicksAreNotLost(t *testing.T) {
	db := setupModelTestDB(t)
	dbConn, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get database connection: %v", err)
	}
	dbConn.SetMaxOpenConns(1)

	link := Link{Code: "concurrent-link", Destination: "https://example.com/concurrent", Active: true}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("failed to create link: %v", err)
	}

	const requests = 25
	var waitGroup sync.WaitGroup
	waitGroup.Add(requests)
	for i := 0; i < requests; i++ {
		go func() {
			defer waitGroup.Done()
			recorder := redirectRequest(t, db, "concurrent-link")
			if recorder.Code != http.StatusFound {
				t.Errorf("expected status %d, got %d", http.StatusFound, recorder.Code)
			}
		}()
	}
	waitGroup.Wait()

	var updated Link
	if err := db.First(&updated, link.ID).Error; err != nil {
		t.Fatalf("failed to load updated link: %v", err)
	}
	if updated.Clicks != requests {
		t.Fatalf("expected %d clicks, got %d", requests, updated.Clicks)
	}
}
