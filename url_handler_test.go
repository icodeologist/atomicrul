package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
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

func TestHandleRedirectionOfShortUrlToLongUrl_Success(t *testing.T) {
	db := setupTestDB(t)

	// Insert a valid URL entry
	url := Url{
		URL:                  "https://google.com",
		ShortID:              "abc123",
		ShortLink:            "http://localhost:3000/abc123",
		ExpirationTime:       time.Now().Add(5 * time.Minute),
		ShortLinkCreatedTime: time.Now(),
	}
	if err := db.Create(&url).Error; err != nil {
		t.Fatalf("failed to insert test url: %v", err)
	}

	// Create a request with mux vars
	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	// Mux doesn't automatically set path params, so we do it manually
	req = mux.SetURLVars(req, map[string]string{"code": "abc123"})

	rr := httptest.NewRecorder()

	HandleRedirectionOfShortUrlToLongUrl(rr, req, db)

	// Expect redirect status
	if rr.Code != http.StatusFound {
		t.Errorf("expected 302 Found, got %d", rr.Code)
	}

	// Expect redirect to correct URL
	location := rr.Header().Get("Location")
	if location != "https://google.com" {
		t.Errorf("expected redirect to https://google.com, got %s", location)
	}

	// Optional: Check if clicks incremented
	var updated Url
	db.First(&updated, url.ID)
	if updated.Clicks != 1 {
		t.Errorf("expected clicks to be 1, got %d", updated.Clicks)
	}
}
