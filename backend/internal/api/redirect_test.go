package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func TestRedirectActiveLink(t *testing.T) {
	db := setupModelTestDB(t)
	link := Link{Code: "active-link", Destination: "https://example.com/current", Active: true}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("failed to create link: %v", err)
	}

	recorder := redirectRequest(t, db, "active-link")

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != link.Destination {
		t.Fatalf("expected redirect to %q, got %q", link.Destination, location)
	}
	var updated Link
	if err := db.First(&updated, link.ID).Error; err != nil {
		t.Fatalf("failed to load updated link: %v", err)
	}
	if updated.Clicks != 1 {
		t.Fatalf("expected one click, got %d", updated.Clicks)
	}
}

func TestRedirectMissingLinkReturnsNotFound(t *testing.T) {
	db := setupModelTestDB(t)
	recorder := redirectRequest(t, db, "missing-link")

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
	if recorder.Header().Get("Location") != "" {
		t.Fatal("missing link must not redirect")
	}
}

func TestRedirectInactiveLinkDoesNotRedirect(t *testing.T) {
	db := setupModelTestDB(t)
	link := Link{Code: "inactive-link", Destination: "https://example.com/inactive", Active: false}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("failed to create link: %v", err)
	}
	if err := db.Model(&link).Update("active", false).Error; err != nil {
		t.Fatalf("failed to deactivate link: %v", err)
	}

	recorder := redirectRequest(t, db, "inactive-link")

	if recorder.Code != http.StatusGone {
		t.Fatalf("expected status %d, got %d", http.StatusGone, recorder.Code)
	}
	if recorder.Header().Get("Location") != "" {
		t.Fatal("inactive link must not redirect")
	}
}

func TestRedirectInvalidOrEmptyCodeReturnsNotFound(t *testing.T) {
	for _, code := range []string{"", "not a valid code"} {
		t.Run(code, func(t *testing.T) {
			db := setupModelTestDB(t)
			recorder := redirectRequest(t, db, code)
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
			}
		})
	}
}

func redirectRequest(t *testing.T, db *gorm.DB, code string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = mux.SetURLVars(request, map[string]string{"code": code})
	recorder := httptest.NewRecorder()
	HandleRedirectionOfShortUrlToLongUrl(recorder, request, db)
	return recorder
}
