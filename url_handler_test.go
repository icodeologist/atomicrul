package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test db : %v", err.Error())
	}

	db.AutoMigrate(&Url{})
	return db
}

var testStore = sessions.NewCookieStore([]byte("test-atomicurl"))

func TestHandleUserUrlsSubmission_Success(t *testing.T) {
	db := setupTestDB(t)

	// Create a fake request body
	body := strings.NewReader("url=https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/create", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create a fake session
	rr := httptest.NewRecorder()
	session, _ := testStore.Get(req, "atomicurl")
	session.Values["authenticated"] = true
	session.Values["userid"] = uint(1)
	session.Save(req, rr)

	// Set the session cookie on request
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	// Call the handler
	handleUserUrlsSumbmission(rr, req, db)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// Optional: check response body contains short url
	if !strings.Contains(rr.Body.String(), "short_url") {
		t.Fatalf("expected short_url in response, got: %s", rr.Body.String())
	}
}
