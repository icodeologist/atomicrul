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

	db.AutoMigrate(Url{}, User{})

	return &Database{
		DB: db,
	}, nil

}

// TODO: make this safer
func (db Database) Migrate(v any) {
	db.DB.AutoMigrate(v)
}
