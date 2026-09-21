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
	config, err := LoadConfig(os.Getenv)
	if err != nil {
		return nil, err
	}
	return ConnectToDatabaseWithConfig(config)
}

func ConnectToDatabaseWithConfig(config AppConfig) (*Database, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	return &Database{
		DB: db,
	}, nil

}

func CloseDatabase(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database handle unavailable: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("database close failed: %w", err)
	}
	return nil
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
