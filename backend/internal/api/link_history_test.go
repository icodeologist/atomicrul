package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func TestGetLinkHistoryReturnsNewestFirstAndMarksCurrentVersion(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	link := createVersionedLink(t, db, user.ID, "history-code", "https://example.com/old", "initial")
	newVersion := LinkVersion{LinkID: link.ID, Destination: "https://example.com/new", Note: "moved"}
	if err := db.Create(&newVersion).Error; err != nil {
		t.Fatalf("failed to create new version: %v", err)
	}
	if err := db.Model(&link).Update("destination", newVersion.Destination).Error; err != nil {
		t.Fatalf("failed to update current destination: %v", err)
	}

	recorder := getHistory(t, db, link.ID, user.ID)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	var response linkHistoryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.LinkID != link.ID || len(response.Versions) != 2 {
		t.Fatalf("unexpected history response: %+v", response)
	}
	if response.Versions[0].Destination != newVersion.Destination || !response.Versions[0].Active {
		t.Fatalf("expected newest version to be current: %+v", response.Versions)
	}
	if response.Versions[1].Destination != "https://example.com/old" || response.Versions[1].Active {
		t.Fatalf("expected initial version to be inactive: %+v", response.Versions)
	}
}

func TestGetLinkHistoryReturnsEmptyList(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	link := Link{Code: "empty-history", Destination: "https://example.com", UserID: user.ID}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("failed to create link: %v", err)
	}

	recorder := getHistory(t, db, link.ID, user.ID)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	var response linkHistoryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Versions == nil || len(response.Versions) != 0 {
		t.Fatalf("expected empty non-null version list, got %#v", response.Versions)
	}
}

func TestGetLinkHistoryUnauthenticated(t *testing.T) {
	db := setupLinkAPITestDB(t)
	request := httptest.NewRequest(http.MethodGet, "/links/1/history", nil)
	request = mux.SetURLVars(request, map[string]string{"id": "1"})
	recorder := httptest.NewRecorder()

	GetLinkHistory(recorder, request, db)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestGetLinkHistoryHidesAnotherUsersLink(t *testing.T) {
	db := setupLinkAPITestDB(t)
	owner := createLinkTestUser(t, db)
	otherUser := User{UserName: "history-other", Email: "history-other@example.com", Password: "password"}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}
	link := createVersionedLink(t, db, owner.ID, "owned-history", "https://example.com", "initial")

	recorder := getHistory(t, db, link.ID, otherUser.ID)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestGetLinkHistoryMissingLink(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	recorder := getHistory(t, db, 999, user.ID)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func getHistory(t *testing.T, db *gorm.DB, linkID, userID uint) *httptest.ResponseRecorder {
	t.Helper()
	request := authenticatedHistoryRequest(t, linkID, userID)
	recorder := httptest.NewRecorder()
	GetLinkHistory(recorder, request, db)
	return recorder
}

func authenticatedHistoryRequest(t *testing.T, linkID, userID uint) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/links/"+strconv.FormatUint(uint64(linkID), 10)+"/history", strings.NewReader(""))
	request = mux.SetURLVars(request, map[string]string{"id": strconv.FormatUint(uint64(linkID), 10)})
	recorder := httptest.NewRecorder()
	session, err := store.Get(request, "atomicurl")
	if err != nil {
		t.Fatalf("failed to get test session: %v", err)
	}
	session.Values["authenticated"] = true
	session.Values["userid"] = userID
	if err := session.Save(request, recorder); err != nil {
		t.Fatalf("failed to save test session: %v", err)
	}
	for _, cookie := range recorder.Result().Cookies() {
		request.AddCookie(cookie)
	}
	return request
}
