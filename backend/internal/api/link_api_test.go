package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestCreateLinkUnauthenticated(t *testing.T) {
	db := setupLinkAPITestDB(t)
	request := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(`{"destination":"https://example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	CreateLink(recorder, request, db)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestCreateLinkSuccessCreatesInitialVersion(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	request := authenticatedLinkRequest(t, `{"title":"Portfolio API","destination":"https://example.com/demo","code":"portfolio-api"}`, user.ID)
	recorder := httptest.NewRecorder()

	CreateLink(recorder, request, db)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	var response linkResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Code != "portfolio-api" || response.Title != "Portfolio API" || response.Destination != "https://example.com/demo" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if !response.Active || response.UserID != user.ID {
		t.Fatalf("unexpected link ownership or active state: %+v", response)
	}

	var link Link
	if err := db.Preload("Versions").First(&link, response.ID).Error; err != nil {
		t.Fatalf("failed to load created link: %v", err)
	}
	if len(link.Versions) != 1 || link.Versions[0].Destination != link.Destination {
		t.Fatalf("expected one initial version matching destination, got %+v", link.Versions)
	}
}

func TestCreateLinkGeneratesCodeAndAllowsMissingTitle(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	request := authenticatedLinkRequest(t, `{"destination":"https://example.com/demo"}`, user.ID)
	recorder := httptest.NewRecorder()

	CreateLink(recorder, request, db)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	var response linkResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Code == "" || response.Title != "" {
		t.Fatalf("expected generated code and empty title, got %+v", response)
	}
	if err := validateLinkCode(response.Code); err != nil {
		t.Fatalf("generated invalid code %q: %v", response.Code, err)
	}
}

func TestCreateLinkRejectsInvalidDestination(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	request := authenticatedLinkRequest(t, `{"destination":"ftp://example.com","code":"valid-code"}`, user.ID)
	recorder := httptest.NewRecorder()

	CreateLink(recorder, request, db)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	var count int64
	db.Model(&Link{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected no link after invalid destination, got %d", count)
	}
}

func TestCreateLinkRejectsDuplicateCustomCode(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	if err := db.Create(&Link{Code: "taken-code", Destination: "https://example.com"}).Error; err != nil {
		t.Fatalf("failed to create existing link: %v", err)
	}
	request := authenticatedLinkRequest(t, `{"destination":"https://example.org","code":"taken-code"}`, user.ID)
	recorder := httptest.NewRecorder()

	CreateLink(recorder, request, db)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, recorder.Code, recorder.Body.String())
	}
}

func TestCreateLinkRejectsInvalidCustomCode(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	request := authenticatedLinkRequest(t, `{"destination":"https://example.com","code":"bad code"}`, user.ID)
	recorder := httptest.NewRecorder()

	CreateLink(recorder, request, db)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestCreateLinkRollsBackWhenInitialVersionFails(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	if err := db.Migrator().DropTable(&LinkVersion{}); err != nil {
		t.Fatalf("failed to remove versions table: %v", err)
	}
	request := authenticatedLinkRequest(t, `{"destination":"https://example.com","code":"rollback-test"}`, user.ID)
	recorder := httptest.NewRecorder()

	CreateLink(recorder, request, db)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	var count int64
	db.Model(&Link{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected transaction rollback, got %d links", count)
	}
}

func setupLinkAPITestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if err := ConfigureSessionStore("test-secret-key-that-is-at-least-32-bytes", false); err != nil {
		t.Fatalf("failed to configure test sessions: %v", err)
	}
	return setupModelTestDB(t)
}

func createLinkTestUser(t *testing.T, db *gorm.DB) User {
	t.Helper()
	user := User{UserName: "link-owner", Email: "link-owner@example.com", Password: "password"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func authenticatedLinkRequest(t *testing.T, body string, userID uint) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/links", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
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
