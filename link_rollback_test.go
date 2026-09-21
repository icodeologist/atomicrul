package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestRollbackLinkCreatesNewVersionAndPreservesHistory(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	link := createVersionedLink(t, db, user.ID, "rollback-code", "https://example.com/first", "initial")
	secondVersion := LinkVersion{LinkID: link.ID, Destination: "https://example.com/second", Note: "deployed"}
	if err := db.Create(&secondVersion).Error; err != nil {
		t.Fatalf("failed to create second version: %v", err)
	}
	if err := db.Model(&link).Update("destination", secondVersion.Destination).Error; err != nil {
		t.Fatalf("failed to set current destination: %v", err)
	}

	var versionsBefore []LinkVersion
	if err := db.Where("link_id = ?", link.ID).Order("id").Find(&versionsBefore).Error; err != nil {
		t.Fatalf("failed to load original history: %v", err)
	}
	request := authenticatedRollbackRequest(t, link.ID, versionsBefore[0].ID, `{"note":"Rolled back because the new deployment was broken"}`, user.ID)
	recorder := httptest.NewRecorder()

	RollbackLink(recorder, request, db)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	var response rollbackLinkResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Link.Code != "rollback-code" || response.Link.Destination != "https://example.com/first" {
		t.Fatalf("unexpected rolled-back link: %+v", response.Link)
	}
	if response.Version.Destination != versionsBefore[0].Destination || response.Version.Note == "" {
		t.Fatalf("unexpected rollback version: %+v", response.Version)
	}

	var versionsAfter []LinkVersion
	if err := db.Where("link_id = ?", link.ID).Order("id").Find(&versionsAfter).Error; err != nil {
		t.Fatalf("failed to load rollback history: %v", err)
	}
	if len(versionsAfter) != 3 {
		t.Fatalf("expected rollback to append a version, got %d versions", len(versionsAfter))
	}
	if versionsAfter[0].Destination != versionsBefore[0].Destination || versionsAfter[0].Note != versionsBefore[0].Note {
		t.Fatalf("original first version was changed: before=%+v after=%+v", versionsBefore[0], versionsAfter[0])
	}
}

func TestRollbackToFirstVersionChangesPublicDestination(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	link := createVersionedLink(t, db, user.ID, "public-rollback", "https://example.com/first", "initial")
	secondVersion := LinkVersion{LinkID: link.ID, Destination: "https://example.com/current", Note: "current"}
	if err := db.Create(&secondVersion).Error; err != nil {
		t.Fatalf("failed to create current version: %v", err)
	}
	if err := db.Model(&link).Update("destination", secondVersion.Destination).Error; err != nil {
		t.Fatalf("failed to set current destination: %v", err)
	}

	var firstVersion LinkVersion
	if err := db.Where("link_id = ?", link.ID).Order("id").First(&firstVersion).Error; err != nil {
		t.Fatalf("failed to load first version: %v", err)
	}
	request := authenticatedRollbackRequest(t, link.ID, firstVersion.ID, `{"note":"restore first version"}`, user.ID)
	recorder := httptest.NewRecorder()
	RollbackLink(recorder, request, db)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	redirectRecorder := redirectRequest(t, db, link.Code)
	if redirectRecorder.Code != http.StatusFound || redirectRecorder.Header().Get("Location") != firstVersion.Destination {
		t.Fatalf("expected public redirect to rolled-back destination, got status=%d location=%q", redirectRecorder.Code, redirectRecorder.Header().Get("Location"))
	}
}

func TestRollbackRejectsInvalidVersionLinkCombination(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	firstLink := createVersionedLink(t, db, user.ID, "first-link", "https://example.com/first", "initial")
	secondLink := createVersionedLink(t, db, user.ID, "second-link", "https://example.com/second", "initial")
	var secondVersion LinkVersion
	if err := db.Where("link_id = ?", secondLink.ID).First(&secondVersion).Error; err != nil {
		t.Fatalf("failed to load second link version: %v", err)
	}
	request := authenticatedRollbackRequest(t, firstLink.ID, secondVersion.ID, `{"note":"invalid cross-link rollback"}`, user.ID)
	recorder := httptest.NewRecorder()

	RollbackLink(recorder, request, db)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestRollbackRequiresNote(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	link := createVersionedLink(t, db, user.ID, "missing-rollback-note", "https://example.com", "initial")
	var version LinkVersion
	if err := db.Where("link_id = ?", link.ID).First(&version).Error; err != nil {
		t.Fatalf("failed to load version: %v", err)
	}
	request := authenticatedRollbackRequest(t, link.ID, version.ID, `{"note":"  "}`, user.ID)
	recorder := httptest.NewRecorder()

	RollbackLink(recorder, request, db)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestRollbackUnauthenticated(t *testing.T) {
	db := setupLinkAPITestDB(t)
	request := httptest.NewRequest(http.MethodPost, "/links/1/versions/1/rollback", strings.NewReader(`{"note":"restore"}`))
	request = mux.SetURLVars(request, map[string]string{"id": "1", "versionID": "1"})
	recorder := httptest.NewRecorder()

	RollbackLink(recorder, request, db)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestRollbackUnauthorizedLinkReturnsNotFound(t *testing.T) {
	db := setupLinkAPITestDB(t)
	owner := createLinkTestUser(t, db)
	otherUser := User{UserName: "rollback-other", Email: "rollback-other@example.com", Password: "password"}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}
	link := createVersionedLink(t, db, owner.ID, "private-rollback", "https://example.com", "initial")
	var version LinkVersion
	if err := db.Where("link_id = ?", link.ID).First(&version).Error; err != nil {
		t.Fatalf("failed to load version: %v", err)
	}
	request := authenticatedRollbackRequest(t, link.ID, version.ID, `{"note":"attempt"}`, otherUser.ID)
	recorder := httptest.NewRecorder()

	RollbackLink(recorder, request, db)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func authenticatedRollbackRequest(t *testing.T, linkID, versionID uint, body string, userID uint) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/links/"+strconv.FormatUint(uint64(linkID), 10)+"/versions/"+strconv.FormatUint(uint64(versionID), 10)+"/rollback", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request = mux.SetURLVars(request, map[string]string{
		"id":        strconv.FormatUint(uint64(linkID), 10),
		"versionID": strconv.FormatUint(uint64(versionID), 10),
	})
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
