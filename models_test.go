package main

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupModelTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := db.AutoMigrate(&Url{}, &User{}, &Link{}, &LinkVersion{}); err != nil {
		t.Fatalf("failed to migrate model test db: %v", err)
	}
	return db
}

func TestLinkCanHaveMultipleVersions(t *testing.T) {
	db := setupModelTestDB(t)
	user := User{UserName: "owner", Email: "owner@example.com", Password: "password"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	link := Link{
		Code:        "abc123",
		Destination: "https://example.com/one",
		UserID:      user.ID,
		Versions: []LinkVersion{
			{Destination: "https://example.com/one", Note: "initial"},
			{Destination: "https://example.com/two", Note: "updated"},
		},
	}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("failed to create link with versions: %v", err)
	}

	var loaded Link
	if err := db.Preload("Versions").First(&loaded, link.ID).Error; err != nil {
		t.Fatalf("failed to load link versions: %v", err)
	}
	if len(loaded.Versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(loaded.Versions))
	}
}

func TestLinkCodesAreUnique(t *testing.T) {
	db := setupModelTestDB(t)
	first := Link{Code: "duplicate", Destination: "https://example.com/one"}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("failed to create first link: %v", err)
	}

	second := Link{Code: "duplicate", Destination: "https://example.com/two"}
	if err := db.Create(&second).Error; err == nil {
		t.Fatal("expected duplicate link code to be rejected")
	}
}

func TestLinksBelongToUsers(t *testing.T) {
	db := setupModelTestDB(t)
	user := User{UserName: "owner", Email: "owner@example.com", Password: "password"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	link := Link{Code: "owned", Destination: "https://example.com", UserID: user.ID}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("failed to create link: %v", err)
	}

	var loaded User
	if err := db.Preload("Links").First(&loaded, user.ID).Error; err != nil {
		t.Fatalf("failed to load user's links: %v", err)
	}
	if len(loaded.Links) != 1 || loaded.Links[0].ID != link.ID {
		t.Fatalf("expected user to own link %d, got %#v", link.ID, loaded.Links)
	}
}

func TestVersionsBelongToLinks(t *testing.T) {
	db := setupModelTestDB(t)
	link := Link{Code: "versioned", Destination: "https://example.com"}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("failed to create link: %v", err)
	}
	version := LinkVersion{LinkID: link.ID, Destination: link.Destination, Note: "initial"}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("failed to create link version: %v", err)
	}

	var loaded LinkVersion
	if err := db.Preload("Link").First(&loaded, version.ID).Error; err != nil {
		t.Fatalf("failed to load version's link: %v", err)
	}
	if loaded.Link.ID != link.ID {
		t.Fatalf("expected version to belong to link %d, got %d", link.ID, loaded.Link.ID)
	}
}
