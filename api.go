package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Request struct {
	LongUrl      string `json:"url"`
	CustomDomain string `json:"custom_domain"`
}

type apiError struct {
	Err string `json:"error"`
}

type apiSuccess struct {
	Short_url string `json:"short_url"`
	Message   any    `json:"message"`
}

func handleUserUrlsSumbmission(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method != http.MethodPost {
		writeJson(w, http.StatusMethodNotAllowed, apiError{Err: "Method not allowed."})
		return
	}

	session, _ := store.Get(r, "atomicurl")
	// check for authentication
	if session.Values["authenticated"] != true {
		writeJson(w, http.StatusUnauthorized, apiError{Err: "User is not authorized.Please login. continue."})
		return
	}

	// get the current logged in users id
	userId := session.Values["userid"]
	if userId == 0 {
		writeJson(w, http.StatusUnauthorized, apiError{Err: "User is not authorized. Please login."})
		return
	}

	longurl := r.FormValue("url")
	if longurl == "" {
		writeJson(w, http.StatusBadRequest, apiError{Err: "Please enter the correct url"})
		return
	}

	// update the database
	url := Url{
		URL:    longurl,
		UserID: userId.(uint),
	}

	result := db.Create(&url)
	if result.Error != nil {
		writeJson(w, http.StatusMethodNotAllowed, apiError{Err: result.Error.Error()})
		return
	}

	// generate a shortid for the id of url

	id := url.ID
	uniqueShortID := GenerateShortIDWithBase62Encoding(id)
	fmt.Println("Short Id ", uniqueShortID)
	url.ShortID = uniqueShortID
	url.ShortLinkCreatedTime = time.Now()
	url.ShortLink = url.Domain + "/" + url.ShortID

	url.ExpirationTime = time.Now().Add(10 * time.Minute)
	// add the domain/shortid and redirect it to main url

	db.Save(&url)
	writeJson(w, 200, apiSuccess{
		Short_url: url.ShortLink,
	})
}

func HandleRedirectionOfShortUrlToLongUrl(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	vars := mux.Vars(r)
	code := vars["code"]
	if code == "" {
		writeJson(w, http.StatusBadRequest, apiError{Err: "code cannot be emtpy."})
		return
	}
	var url Url
	result := db.Where("short_id=?", code).First(&url)
	if result.Error != nil {
		writeJson(w, http.StatusBadRequest, apiError{Err: fmt.Sprintf("%v\n", result.Error.Error())})
		return
	}

	// chekc if  the link is expired
	if time.Now().After(url.ExpirationTime) {
		// link has expired
		writeJson(w, http.StatusNotAcceptable, apiError{Err: "Your link has expired."})
		return
	}

	// update the clicks
	db.Model(&url).Update("clicks", url.Clicks+1)
	fmt.Println("url ->", url)

	http.Redirect(w, r, url.URL, 302)
}

func writeJson(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

func ReverseString(s string) string {
	r := []rune(s)
	n := len(r)
	for i := 0; i < n/2; i++ {
		fmt.Println(string(n - 1 - i))
		r[i], r[n-1-i] = r[n-1-i], r[i]
	}
	return string(r)
}
