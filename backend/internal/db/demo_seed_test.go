package db

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
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

func TestSeedDemoDataIsIdempotent(t *testing.T) {
	db := setupModelTestDB(t)

	if err := SeedDemoData(db); err != nil {
		t.Fatalf("failed to seed demo data: %v", err)
	}
	if err := SeedDemoData(db); err != nil {
		t.Fatalf("failed to repeat demo seed: %v", err)
	}

	var user User
	if err := db.Where("user_name = ?", demoUsername).First(&user).Error; err != nil {
		t.Fatalf("failed to find demo user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(demoPassword)); err != nil {
		t.Fatalf("demo password was not seeded correctly: %v", err)
	}

	var linkCount int64
	if err := db.Model(&Link{}).Where("user_id = ?", user.ID).Count(&linkCount).Error; err != nil {
		t.Fatalf("failed to count demo links: %v", err)
	}
	if linkCount != int64(len(demoLinks)) {
		t.Fatalf("expected %d demo links, got %d", len(demoLinks), linkCount)
	}

	var portfolio Link
	if err := db.Preload("Versions").Where("code = ?", "portfolio").First(&portfolio).Error; err != nil {
		t.Fatalf("failed to load demo portfolio: %v", err)
	}
	if len(portfolio.Versions) != 2 || portfolio.Clicks != 42 {
		t.Fatalf("expected portfolio history and clicks, got %#v", portfolio)
	}
}
