package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestDashboardReturnsOwnedLinksSortedWithVersionCounts(t *testing.T) {
	db := setupLinkAPITestDB(t)
	owner := createLinkTestUser(t, db)
	other := createLinkTestUserWithDetails(t, db, "dashboard-other", "dashboard-other@example.com")

	older := createVersionedLink(t, db, owner.ID, "older-code", "https://example.com/older", "initial")
	newer := createVersionedLink(t, db, owner.ID, "newer-code", "https://example.com/newer", "initial")
	if err := db.Model(&older).Updates(map[string]any{"updated_at": time.Now().Add(-time.Hour)}).Error; err != nil {
		t.Fatalf("failed to set older update time: %v", err)
	}
	if err := db.Model(&newer).Updates(map[string]any{"updated_at": time.Now()}).Error; err != nil {
		t.Fatalf("failed to set newer update time: %v", err)
	}
	newerVersion := LinkVersion{LinkID: newer.ID, Destination: "https://example.com/newer-2", Note: "updated"}
	if err := db.Create(&newerVersion).Error; err != nil {
		t.Fatalf("failed to create newer version: %v", err)
	}
	if err := db.Model(&newer).Update("destination", newerVersion.Destination).Error; err != nil {
		t.Fatalf("failed to update newer destination: %v", err)
	}
	if err := db.Model(&newer).Updates(map[string]any{"updated_at": time.Now()}).Error; err != nil {
		t.Fatalf("failed to refresh newer update time: %v", err)
	}
	inactive := Link{Code: "inactive-dashboard", Title: "Inactive", Destination: "https://example.com/inactive", Active: true, UserID: owner.ID}
	if err := db.Create(&inactive).Error; err != nil {
		t.Fatalf("failed to create inactive link: %v", err)
	}
	if err := db.Model(&inactive).Update("active", false).Error; err != nil {
		t.Fatalf("failed to deactivate link: %v", err)
	}
	if err := db.Model(&inactive).Updates(map[string]any{"updated_at": time.Now().Add(-2 * time.Hour)}).Error; err != nil {
		t.Fatalf("failed to set inactive update time: %v", err)
	}
	otherLink := Link{Code: "other-dashboard", Destination: "https://example.com/other", UserID: other.ID}
	if err := db.Create(&otherLink).Error; err != nil {
		t.Fatalf("failed to create other user's link: %v", err)
	}

	recorder := dashboardRequest(t, db, owner.ID)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	var response dashboardResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode dashboard response: %v", err)
	}
	if len(response.Links) != 3 {
		t.Fatalf("expected only owner's 3 links, got %d", len(response.Links))
	}
	if response.Links[0].Code != "newer-code" || response.Links[1].Code != "older-code" || response.Links[2].Code != "inactive-dashboard" {
		t.Fatalf("expected links sorted by update time, got %#v", response.Links)
	}
	if response.Links[0].VersionCount != 2 || response.Links[0].Destination != newerVersion.Destination {
		t.Fatalf("expected current destination and version count, got %#v", response.Links[0])
	}
	if response.Links[2].Active {
		t.Fatal("expected inactive link to be represented as inactive")
	}
	if response.Links[0].ShortURL != "http://dashboard.example/newer-code" {
		t.Fatalf("unexpected short URL: %q", response.Links[0].ShortURL)
	}
}

func TestDashboardUnauthenticated(t *testing.T) {
	db := setupLinkAPITestDB(t)
	request := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	recorder := httptest.NewRecorder()

	ShowUserDashBoard(recorder, request, db)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestDashboardEmptyReturnsArray(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	recorder := dashboardRequest(t, db, user.ID)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	var response dashboardResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode dashboard response: %v", err)
	}
	if response.Links == nil || len(response.Links) != 0 {
		t.Fatalf("expected empty links array, got %#v", response.Links)
	}
}

func dashboardRequest(t *testing.T, db *gorm.DB, userID uint) *httptest.ResponseRecorder {
	t.Helper()
	request := requestWithSession(t, map[string]any{"authenticated": true, "userid": userID})
	request.URL.Path = "/dashboard"
	request.Host = "dashboard.example"
	recorder := httptest.NewRecorder()
	ShowUserDashBoard(recorder, request, db)
	return recorder
}
