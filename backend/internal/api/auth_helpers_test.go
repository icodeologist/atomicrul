package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/gorm"
)

func TestRequireAuthenticatedUserRejectsMissingSession(t *testing.T) {
	if err := ConfigureSessionStore("test-secret-key-that-is-at-least-32-bytes", false); err != nil {
		t.Fatalf("failed to configure test sessions: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/links/1/history", nil)
	recorder := httptest.NewRecorder()

	if _, ok := requireAuthenticatedUser(recorder, request); ok {
		t.Fatal("expected missing session to be rejected")
	}
	assertAPIError(t, recorder, http.StatusUnauthorized, "Please log in.")
}

func TestRequireAuthenticatedUserRejectsUnauthenticatedSession(t *testing.T) {
	request := requestWithSession(t, map[string]any{"authenticated": false, "userid": uint(1)})
	recorder := httptest.NewRecorder()

	if _, ok := requireAuthenticatedUser(recorder, request); ok {
		t.Fatal("expected unauthenticated session to be rejected")
	}
	assertAPIError(t, recorder, http.StatusUnauthorized, "Please log in.")
}

func TestRequireAuthenticatedUserRejectsMalformedUserID(t *testing.T) {
	request := requestWithSession(t, map[string]any{"authenticated": true, "userid": "not-a-number"})
	recorder := httptest.NewRecorder()

	if _, ok := requireAuthenticatedUser(recorder, request); ok {
		t.Fatal("expected malformed user ID to be rejected")
	}
	assertAPIError(t, recorder, http.StatusUnauthorized, "Please log in.")
}

func TestFindOwnedLinkHidesOtherUsersAndMissingLinks(t *testing.T) {
	db := setupLinkAPITestDB(t)
	owner := createLinkTestUser(t, db)
	other := createLinkTestUserWithDetails(t, db, "helper-other", "helper-other@example.com")
	link := Link{Code: "helper-owned", Destination: "https://example.com", UserID: owner.ID}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("failed to create link: %v", err)
	}

	if _, err := findOwnedLink(db, other.ID, link.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected another user's link to appear missing, got %v", err)
	}
	if _, err := findOwnedLink(db, owner.ID, 999); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected nonexistent link to appear missing, got %v", err)
	}
}

func TestWriteAPIErrorUsesStableJSONShape(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeAPIError(recorder, http.StatusNotFound, "Link not found.")
	assertAPIError(t, recorder, http.StatusNotFound, "Link not found.")
}

func requestWithSession(t *testing.T, values map[string]any) *http.Request {
	t.Helper()
	if store == nil {
		if err := ConfigureSessionStore("test-secret-key-that-is-at-least-32-bytes", false); err != nil {
			t.Fatalf("failed to configure test sessions: %v", err)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	session, err := store.Get(request, "atomicurl")
	if err != nil {
		t.Fatalf("failed to get test session: %v", err)
	}
	for key, value := range values {
		session.Values[key] = value
	}
	if err := session.Save(request, recorder); err != nil {
		t.Fatalf("failed to save test session: %v", err)
	}
	for _, cookie := range recorder.Result().Cookies() {
		request.AddCookie(cookie)
	}
	return request
}

func assertAPIError(t *testing.T, recorder *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("expected status %d, got %d", status, recorder.Code)
	}
	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode API error: %v", err)
	}
	if response["error"] != message {
		t.Fatalf("expected error %q, got %#v", message, response)
	}
}

func createLinkTestUserWithDetails(t *testing.T, db *gorm.DB, username, email string) User {
	t.Helper()
	user := User{UserName: username, Email: email, Password: "password"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}
