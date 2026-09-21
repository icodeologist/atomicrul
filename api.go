package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type apiError struct {
	Err string `json:"error"`
}

type apiSuccess struct {
	Short_url string `json:"short_url"`
	Success   any    `json:"message"`
}

func handleUserUrlsSumbmission(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method != http.MethodPost {
		writeJson(w, http.StatusMethodNotAllowed, apiError{Err: "Method not allowed."})
		return
	}

	session, ok := getSession(w, r)
	if !ok {
		return
	}
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

	// add the domain/shortid and redirect it to main url

	db.Save(&url)
	writeJson(w, 200, apiSuccess{
		Short_url: url.ShortLink,
	})
}

func HandleRedirectionOfShortUrlToLongUrl(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method != http.MethodGet {
		writeJson(w, http.StatusMethodNotAllowed, apiError{Err: "Method not allowed."})
		return
	}

	code := strings.TrimSpace(mux.Vars(r)["code"])
	if code == "" {
		writeJson(w, http.StatusNotFound, apiError{Err: "Link not found."})
		return
	}

	var link Link
	result := db.Where("code = ?", code).First(&link)
	if result.Error == nil {
		if !link.Active {
			writeJson(w, http.StatusGone, apiError{Err: "This link is inactive."})
			return
		}

		clickResult := db.Model(&Link{}).
			Where("id = ? AND active = ?", link.ID, true).
			UpdateColumn("clicks", gorm.Expr("clicks + ?", 1))
		if clickResult.Error != nil {
			writeJson(w, http.StatusInternalServerError, apiError{Err: "Could not record link click."})
			return
		}
		if clickResult.RowsAffected == 0 {
			writeJson(w, http.StatusGone, apiError{Err: "This link is inactive."})
			return
		}

		http.Redirect(w, r, link.Destination, http.StatusFound)
		return
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		writeJson(w, http.StatusInternalServerError, apiError{Err: "Could not look up link."})
		return
	}

	// Legacy Url records remain available until the explicit data migration.
	var url Url
	result = db.Where("short_id = ?", code).First(&url)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		writeJson(w, http.StatusNotFound, apiError{Err: "Link not found."})
		return
	}
	if result.Error != nil {
		writeJson(w, http.StatusInternalServerError, apiError{Err: "Could not look up link."})
		return
	}

	clickResult := db.Model(&Url{}).
		Where("id = ?", url.ID).
		UpdateColumn("clicks", gorm.Expr("clicks + ?", 1))
	if clickResult.Error != nil {
		writeJson(w, http.StatusInternalServerError, apiError{Err: "Could not record link click."})
		return
	}
	if clickResult.RowsAffected == 0 {
		writeJson(w, http.StatusNotFound, apiError{Err: "Link not found."})
		return
	}

	http.Redirect(w, r, url.URL, http.StatusFound)
}

func writeJson(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}
