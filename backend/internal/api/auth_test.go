package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func TestConfigureSessionStoreRejectsShortSecret(t *testing.T) {
	if err := ConfigureSessionStore("too-short", false); err == nil {
		t.Fatal("expected a short session secret to be rejected")
	}
}

func TestRegisterDoesNotExposePasswordHash(t *testing.T) {
	db := setupAuthTestDB(t)
	body := strings.NewReader("username=denzil&email=denzil%40example.com&password=correct-horse")
	req := httptest.NewRequest(http.MethodPost, "/register", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	Register(recorder, req, db)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	var user User
	if err := db.First(&user).Error; err != nil {
		t.Fatalf("expected registered user: %v", err)
	}
	if strings.Contains(recorder.Body.String(), user.Password) || strings.Contains(recorder.Body.String(), "correct-horse") {
		t.Fatalf("registration response exposed password data: %s", recorder.Body.String())
	}
}

func TestLoginUsesGenericErrorForWrongPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	registerBody := strings.NewReader("username=denzil&email=denzil%40example.com&password=correct-horse")
	registerRequest := httptest.NewRequest(http.MethodPost, "/register", registerBody)
	registerRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	registerRecorder := httptest.NewRecorder()
	Register(registerRecorder, registerRequest, db)
	if registerRecorder.Code != http.StatusCreated {
		t.Fatalf("failed to create login test user: %s", registerRecorder.Body.String())
	}

	loginBody := strings.NewReader("username=denzil&password=wrong-password")
	req := httptest.NewRequest(http.MethodPost, "/login", loginBody)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	Login(recorder, req, db)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Invalid username or password") {
		t.Fatalf("expected generic login error, got: %s", recorder.Body.String())
	}
}

func TestLoginRendersBrowserForm(t *testing.T) {
	db := setupAuthTestDB(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/login", nil)

	Login(recorder, request, db)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("expected HTML content type, got %q", contentType)
	}
	if !strings.Contains(recorder.Body.String(), `action="/login"`) {
		t.Fatalf("expected login form, got: %s", recorder.Body.String())
	}
}
