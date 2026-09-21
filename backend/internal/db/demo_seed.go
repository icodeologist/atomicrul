package db

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	demoUsername = "demo"
	demoEmail    = "demo@example.com"
	demoPassword = "demo-password"
)

type demoLink struct {
	code        string
	title       string
	destination string
	clicks      int
	active      bool
	versions    []demoVersion
}

type demoVersion struct {
	destination string
	note        string
}

var demoLinks = []demoLink{
	{
		code:        "portfolio",
		title:       "Portfolio",
		destination: "https://example.com/portfolio",
		clicks:      42,
		active:      true,
		versions: []demoVersion{
			{destination: "https://example.com/old-portfolio", note: "Original destination"},
			{destination: "https://example.com/portfolio", note: "Updated portfolio URL"},
		},
	},
	{
		code:        "docs",
		title:       "Project documentation",
		destination: "https://go.dev/doc/",
		clicks:      17,
		active:      true,
		versions: []demoVersion{
			{destination: "https://go.dev/doc/", note: "Initial destination"},
		},
	},
	{
		code:        "old-campaign",
		title:       "Old campaign",
		destination: "https://example.com/old-campaign",
		clicks:      8,
		active:      false,
		versions: []demoVersion{
			{destination: "https://example.com/old-campaign", note: "Campaign ended"},
		},
	},
}

// SeedDemoData creates an idempotent local demo account and links. It is only
// called when explicitly enabled through ATOMICURL_SEED_DEMO=true.
func SeedDemoData(db *gorm.DB) error {
	password, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash demo password: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var user User
		result := tx.Where("user_name = ?", demoUsername).First(&user)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			user = User{UserName: demoUsername, Email: demoEmail, Password: string(password)}
			if err := tx.Create(&user).Error; err != nil {
				return fmt.Errorf("create demo user: %w", err)
			}
		} else if result.Error != nil {
			return fmt.Errorf("find demo user: %w", result.Error)
		}

		for _, seed := range demoLinks {
			var link Link
			result := tx.Where("code = ?", seed.code).First(&link)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				link = Link{
					Code:        seed.code,
					Title:       seed.title,
					Destination: seed.destination,
					Active:      seed.active,
					Clicks:      seed.clicks,
					UserID:      user.ID,
				}
				if err := tx.Create(&link).Error; err != nil {
					return fmt.Errorf("create demo link %q: %w", seed.code, err)
				}
				// GORM may omit a false zero value when the column has a default.
				// Set it explicitly so the inactive demo link is really inactive.
				if err := tx.Model(&link).Update("active", seed.active).Error; err != nil {
					return fmt.Errorf("set demo link status %q: %w", seed.code, err)
				}
				for _, seedVersion := range seed.versions {
					if err := tx.Create(&LinkVersion{LinkID: link.ID, Destination: seedVersion.destination, Note: seedVersion.note}).Error; err != nil {
						return fmt.Errorf("create demo history for %q: %w", seed.code, err)
					}
				}
				continue
			}
			if result.Error != nil {
				return fmt.Errorf("find demo link %q: %w", seed.code, result.Error)
			}
			if err := tx.Model(&link).Update("active", seed.active).Error; err != nil {
				return fmt.Errorf("refresh demo link status %q: %w", seed.code, err)
			}
		}
		return nil
	})
}
