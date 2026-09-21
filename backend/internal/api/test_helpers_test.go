package api

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
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}
