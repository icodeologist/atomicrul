package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func main() {
	if err := ConfigureSessionStore(os.Getenv("SECRETKEY"), os.Getenv("APP_ENV") == "production"); err != nil {
		log.Fatal(err)
	}

	db, err := SetUpDb()
	if err != nil {
		log.Fatal(err.Error())
	}

	r := mux.NewRouter()
	r.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		Register(w, r, db)
	}).Methods("POST")

	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		Login(w, r, db)
	}).Methods("POST")

	rateLimiter := RateLimiterMiddleware(rate.Limit(5), 10)

	r.Handle("/create", rateLimiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleUserUrlsSumbmission(w, r, db)
	}))).Methods(http.MethodPost)
	r.HandleFunc("/links", func(w http.ResponseWriter, r *http.Request) {
		CreateLink(w, r, db)
	}).Methods(http.MethodPost)

	r.HandleFunc("/logout", Logout).Methods(http.MethodPost)
	r.HandleFunc("/greetme", func(w http.ResponseWriter, r *http.Request) {
		GreetIn(w, r, db)
	}).Methods(http.MethodGet)

	r.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		ShowUserDashBoard(w, r, db)
	}).Methods(http.MethodGet)

	r.HandleFunc("/remake_links", func(w http.ResponseWriter, r *http.Request) {
		RemakeExpiredLinks(w, r, db)
	}).Methods(http.MethodPost)
	r.HandleFunc("/{code}", func(w http.ResponseWriter, r *http.Request) {
		HandleRedirectionOfShortUrlToLongUrl(w, r, db)
	}).Methods(http.MethodGet)

	fmt.Println("Server running on port 3000")
	err = http.ListenAndServe(":3000", r)
	if err != nil {
		fmt.Println(err)
	}
}

func SetUpDb() (*gorm.DB, error) {
	databse, err := ConnectToDatabase()
	if err != nil {
		return nil, err
	}
	db := databse.DB

	if err := MigrateDatabase(db); err != nil {
		return nil, err
	}
	return db, nil
}
