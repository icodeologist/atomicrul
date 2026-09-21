package db

import (
	"fmt"
	"github.com/icodeologist/atomicurl/internal/config"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
)

type Database struct {
	DB *gorm.DB
}

func ConnectToDatabase() (*Database, error) {
	appConfig, err := config.LoadConfig(os.Getenv)
	if err != nil {
		return nil, err
	}
	return ConnectToDatabaseWithConfig(appConfig)
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

func SetUpDatabaseWithConfig(config AppConfig) (*gorm.DB, error) {
	database, err := ConnectToDatabaseWithConfig(config)
	if err != nil {
		return nil, err
	}
	if err := MigrateDatabase(database.DB); err != nil {
		_ = CloseDatabase(database.DB)
		return nil, err
	}
	return database.DB, nil
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
	// Migrate referenced tables first. PostgreSQL rejects the foreign keys on
	// Url, Link, and LinkVersion if their parent tables do not exist yet.
	if err := db.AutoMigrate(&User{}, &Link{}, &LinkVersion{}, &Url{}); err != nil {
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
