package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/gorm"
)

func TestPermanentLinkLifecycleIntegration(t *testing.T) {
	db := setupLinkAPITestDB(t)
	user := createLinkTestUser(t, db)
	const (
		code         = "integration-link"
		destinationA = "https://example.com/version-a"
		destinationB = "https://example.com/version-b"
	)

	createRequest := authenticatedLinkRequest(t, `{"title":"Integration link","destination":"`+destinationA+`","code":"`+code+`"}`, user.ID)
	createRecorder := httptest.NewRecorder()
	CreateLink(createRecorder, createRequest, db)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected link creation status %d, got %d: %s", http.StatusCreated, createRecorder.Code, createRecorder.Body.String())
	}
	var created linkResponse
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode creation response: %v", err)
	}
	if created.Code != code {
		t.Fatalf("expected code %q, got %q", code, created.Code)
	}

	assertRedirectsTo(t, db, code, destinationA)

	updateRequest := authenticatedUpdateLinkRequest(t, created.ID, `{"destination":"`+destinationB+`","note":"Moved to version B"}`, user.ID)
	updateRecorder := httptest.NewRecorder()
	UpdateLink(updateRecorder, updateRequest, db)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected update status %d, got %d: %s", http.StatusOK, updateRecorder.Code, updateRecorder.Body.String())
	}

	assertRedirectsTo(t, db, code, destinationB)

	historyRecorder := getHistory(t, db, created.ID, user.ID)
	if historyRecorder.Code != http.StatusOK {
		t.Fatalf("expected history status %d, got %d", http.StatusOK, historyRecorder.Code)
	}
	var history linkHistoryResponse
	if err := json.Unmarshal(historyRecorder.Body.Bytes(), &history); err != nil {
		t.Fatalf("failed to decode history response: %v", err)
	}
	if len(history.Versions) != 2 {
		t.Fatalf("expected 2 versions before rollback, got %d", len(history.Versions))
	}
	firstVersionID := history.Versions[1].ID
	if history.Versions[1].Destination != destinationA || history.Versions[0].Destination != destinationB {
		t.Fatalf("unexpected pre-rollback history: %+v", history.Versions)
	}

	rollbackRequest := authenticatedRollbackRequest(t, created.ID, firstVersionID, `{"note":"Rolled back to version A"}`, user.ID)
	rollbackRecorder := httptest.NewRecorder()
	RollbackLink(rollbackRecorder, rollbackRequest, db)
	if rollbackRecorder.Code != http.StatusOK {
		t.Fatalf("expected rollback status %d, got %d: %s", http.StatusOK, rollbackRecorder.Code, rollbackRecorder.Body.String())
	}
	var rollback rollbackLinkResponse
	if err := json.Unmarshal(rollbackRecorder.Body.Bytes(), &rollback); err != nil {
		t.Fatalf("failed to decode rollback response: %v", err)
	}
	if rollback.Link.Code != code || rollback.Link.Destination != destinationA {
		t.Fatalf("rollback changed code or destination unexpectedly: %+v", rollback.Link)
	}

	assertRedirectsTo(t, db, code, destinationA)

	finalHistoryRecorder := getHistory(t, db, created.ID, user.ID)
	var finalHistory linkHistoryResponse
	if err := json.Unmarshal(finalHistoryRecorder.Body.Bytes(), &finalHistory); err != nil {
		t.Fatalf("failed to decode final history response: %v", err)
	}
	if len(finalHistory.Versions) != 3 {
		t.Fatalf("expected rollback to append a third version, got %d", len(finalHistory.Versions))
	}
	if finalHistory.Versions[0].Destination != destinationA || !finalHistory.Versions[0].Active {
		t.Fatalf("expected rollback version to be current: %+v", finalHistory.Versions[0])
	}

	var finalLink Link
	if err := db.First(&finalLink, created.ID).Error; err != nil {
		t.Fatalf("failed to load final link: %v", err)
	}
	if finalLink.Code != code {
		t.Fatalf("expected code to remain %q, got %q", code, finalLink.Code)
	}
	if finalLink.Clicks != 3 {
		t.Fatalf("expected 3 recorded clicks, got %d", finalLink.Clicks)
	}
}

func assertRedirectsTo(t *testing.T, db *gorm.DB, code, destination string) {
	t.Helper()
	recorder := redirectRequest(t, db, code)
	if recorder.Code != http.StatusFound {
		t.Fatalf("expected redirect status %d, got %d", http.StatusFound, recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != destination {
		t.Fatalf("expected redirect to %q, got %q", destination, location)
	}
}
