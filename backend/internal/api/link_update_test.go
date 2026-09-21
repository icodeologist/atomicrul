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

func TestUpdateLinkSuccessPreservesCodeAndHistory(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	link := createVersionedLink(t, db, user.ID, "stable-code", "https://example.com/old", "initial")
	request := authenticatedUpdateLinkRequest(t, link.ID, `{"destination":"https://new-host.example/demo","note":"Moved the demo to the new deployment"}`, user.ID)
	recorder := httptest.NewRecorder()

	UpdateLink(recorder, request, db)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	var response updateLinkResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Link.Code != "stable-code" || response.Link.Destination != "https://new-host.example/demo" {
		t.Fatalf("unexpected updated link: %+v", response.Link)
	}
	if response.Version.LinkID != link.ID || response.Version.Destination != response.Link.Destination || response.Version.Note == "" {
		t.Fatalf("unexpected new version: %+v", response.Version)
	}

	var versions []LinkVersion
	if err := db.Where("link_id = ?", link.ID).Order("id").Find(&versions).Error; err != nil {
		t.Fatalf("failed to load link history: %v", err)
	}
	if len(versions) != 2 || versions[0].Destination != "https://example.com/old" || versions[0].Note != "initial" {
		t.Fatalf("expected original version to remain intact, got %+v", versions)
	}
}

func TestUpdateLinkUnauthenticated(t *testing.T) {
	db := setupLinkAPITestDB(t)
	request := httptest.NewRequest(http.MethodPatch, "/links/1", strings.NewReader(`{"destination":"https://example.com/new","note":"move"}`))
	request = mux.SetURLVars(request, map[string]string{"id": "1"})
	recorder := httptest.NewRecorder()

	UpdateLink(recorder, request, db)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestUpdateLinkHidesOwnership(t *testing.T) {
	db := setupLinkAPITestDB(t)
	owner := createLinkTestUser(t, db)
	otherUser := User{UserName: "other-owner", Email: "other-owner@example.com", Password: "password"}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}
	link := createVersionedLink(t, db, owner.ID, "private-code", "https://example.com/old", "initial")
	request := authenticatedUpdateLinkRequest(t, link.ID, `{"destination":"https://example.com/new","note":"attempt"}`, otherUser.ID)
	recorder := httptest.NewRecorder()

	UpdateLink(recorder, request, db)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestUpdateLinkMissingReturnsNotFound(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	request := authenticatedUpdateLinkRequest(t, 999, `{"destination":"https://example.com/new","note":"move"}`, user.ID)
	recorder := httptest.NewRecorder()

	UpdateLink(recorder, request, db)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestUpdateLinkRejectsInvalidDestination(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	link := createVersionedLink(t, db, user.ID, "invalid-destination", "https://example.com/old", "initial")
	request := authenticatedUpdateLinkRequest(t, link.ID, `{"destination":"ftp://example.com/new","note":"move"}`, user.ID)
	recorder := httptest.NewRecorder()

	UpdateLink(recorder, request, db)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestUpdateLinkRequiresNote(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	link := createVersionedLink(t, db, user.ID, "missing-note", "https://example.com/old", "initial")
	request := authenticatedUpdateLinkRequest(t, link.ID, `{"destination":"https://example.com/new","note":"  "}`, user.ID)
	recorder := httptest.NewRecorder()

	UpdateLink(recorder, request, db)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestUpdateLinkRejectsUnchangedDestination(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	link := createVersionedLink(t, db, user.ID, "unchanged", "https://example.com/old", "initial")
	request := authenticatedUpdateLinkRequest(t, link.ID, `{"destination":"https://example.com/old","note":"same destination"}`, user.ID)
	recorder := httptest.NewRecorder()

	UpdateLink(recorder, request, db)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	var count int64
	db.Model(&LinkVersion{}).Where("link_id = ?", link.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected unchanged destination to create no version, got %d", count)
	}
}

func TestUpdateLinkRollsBackWhenVersionCreationFails(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	link := createVersionedLink(t, db, user.ID, "rollback-update", "https://example.com/old", "initial")
	if err := db.Migrator().DropTable(&LinkVersion{}); err != nil {
		t.Fatalf("failed to remove versions table: %v", err)
	}
	request := authenticatedUpdateLinkRequest(t, link.ID, `{"destination":"https://example.com/new","note":"move"}`, user.ID)
	recorder := httptest.NewRecorder()

	UpdateLink(recorder, request, db)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	var unchanged Link
	if err := db.First(&unchanged, link.ID).Error; err != nil {
		t.Fatalf("failed to load link after rollback: %v", err)
	}
	if unchanged.Destination != "https://example.com/old" {
		t.Fatalf("expected destination rollback, got %q", unchanged.Destination)
	}
}

func createVersionedLink(t *testing.T, db *gorm.DB, userID uint, code, destination, note string) Link {
	t.Helper()
	link := Link{Code: code, Destination: destination, Active: true, UserID: userID}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("failed to create link: %v", err)
	}
	version := LinkVersion{LinkID: link.ID, Destination: destination, Note: note}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("failed to create initial version: %v", err)
	}
	return link
}

func authenticatedUpdateLinkRequest(t *testing.T, linkID uint, body string, userID uint) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPatch, "/links/"+strconv.FormatUint(uint64(linkID), 10), strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
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
