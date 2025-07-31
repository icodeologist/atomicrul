package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func main() {
	//FIXME: add better error handling
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

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleUserUrlsSumbmission(w, r, db)
	})

	r.HandleFunc("/logout", Logout)
	r.HandleFunc("/greetme", func(w http.ResponseWriter, r *http.Request) {
		GreetIn(w, r, db)
	})

	r.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		ShowUserDashBoard(w, r, db)
	})

	r.HandleFunc("/remake_links", func(w http.ResponseWriter, r *http.Request) {
		RemakeExpiredLinks(w, r, db)
	})
	r.HandleFunc("/{code}", func(w http.ResponseWriter, r *http.Request) {
		HandleRedirectionOfShortUrlToLongUrl(w, r, db)
	})

	fmt.Println("Server running in port:8000")
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

	err = db.AutoMigrate(&Url{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
