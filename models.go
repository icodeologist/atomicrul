package main

import (
	"time"

	_ "gorm.io/gorm"
)

type User struct {
	ID        uint   `gorm:"primaryKey"`
	UserName  string `gorm:"not null;size:100"`
	Email     string `gorm:"uniqueIndex;not null;size:100"`
	Password  string `gorm:"not null"`
	Urls      []Url  `gorm:"foreignKey:UserID"`
	Links     []Link `gorm:"foreignKey:UserID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TODO: Refactor these fields
type Url struct {
	ID  uint   `gorm:"primaryKey;autoIncrement"`
	URL string // user sends this link via post req
	// initially all short Id fields are empty
	// later they will be filled
	ShortID              string
	ShortLink            string
	Domain               string `gorm:"default:'http://localhost:3000'"`
	ShortLinkCreatedTime time.Time
	ExpirationTime       time.Time
	UserID               uint
	Clicks               int
}

type Link struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	Code        string `gorm:"uniqueIndex;not null"`
	Title       string
	Destination string        `gorm:"not null"`
	Active      bool          `gorm:"not null;default:true"`
	Clicks      int           `gorm:"not null;default:0"`
	UserID      uint          `gorm:"index;not null"`
	User        User          `gorm:"foreignKey:UserID"`
	Versions    []LinkVersion `gorm:"foreignKey:LinkID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type LinkVersion struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	LinkID      uint   `gorm:"index;not null"`
	Link        Link   `gorm:"foreignKey:LinkID"`
	Destination string `gorm:"not null"`
	Note        string
	CreatedAt   time.Time
}
