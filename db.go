package main

import (
	"fmt"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
)

type Database struct {
	DB *gorm.DB
}

func ConnectToDatabase() (*Database, error) {
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	user := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	dbname := os.Getenv("DBNAME")
	dsn := fmt.Sprintf("host=%s port=%v user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("Database connection error : %v\n", err)
	}
	fmt.Println("Database connected successfully.")

	return &Database{
		DB: db,
	}, nil

}

func MigrateDatabase(db *gorm.DB) error {
	if err := db.AutoMigrate(&Url{}, &User{}, &Link{}, &LinkVersion{}); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}
	return nil
}

func (db Database) Migrate(v any) error {
	if err := db.DB.AutoMigrate(v); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}
	return nil
}
