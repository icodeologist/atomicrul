package main

import (
	"encoding/json"
	"errors"
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
	ShortURL string `json:"short_url"`
	Success  any    `json:"message"`
}

func handleUserUrlsSumbmission(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method != http.MethodPost {
		writeJson(w, http.StatusMethodNotAllowed, apiError{Err: "Method not allowed."})
		return
	}

	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	longurl := r.FormValue("url")
	destination, err := ValidateAndNormalizeURL(longurl)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	// update the database
	url := Url{
		URL:    destination,
		UserID: userID,
	}

	result := db.Create(&url)
	if result.Error != nil {
		writeAPIError(w, http.StatusInternalServerError, "Could not create legacy link.")
		return
	}

	// generate a shortid for the id of url

	id := url.ID
	uniqueShortID := GenerateShortIDWithBase62Encoding(id)
	url.ShortID = uniqueShortID
	url.ShortLinkCreatedTime = time.Now()
	url.ShortLink = url.Domain + "/" + url.ShortID

	// add the domain/shortid and redirect it to main url

	if err := db.Save(&url).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Could not save legacy link.")
		return
	}
	writeJson(w, http.StatusOK, apiSuccess{
		ShortURL: url.ShortLink,
	})
}

func HandleRedirectionOfShortUrlToLongUrl(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "Method not allowed.")
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
			writeAPIError(w, http.StatusGone, "This link is inactive.")
			return
		}

		clickResult := db.Model(&Link{}).
			Where("id = ? AND active = ?", link.ID, true).
			UpdateColumn("clicks", gorm.Expr("clicks + ?", 1))
		if clickResult.Error != nil {
			writeAPIError(w, http.StatusInternalServerError, "Could not record link click.")
			return
		}
		if clickResult.RowsAffected == 0 {
			writeAPIError(w, http.StatusGone, "This link is inactive.")
			return
		}

		http.Redirect(w, r, link.Destination, http.StatusFound)
		return
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		writeAPIError(w, http.StatusInternalServerError, "Could not look up link.")
		return
	}

	// Legacy Url records remain available until the explicit data migration.
	var url Url
	result = db.Where("short_id = ?", code).First(&url)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		writeAPIError(w, http.StatusNotFound, "Link not found.")
		return
	}
	if result.Error != nil {
		writeAPIError(w, http.StatusInternalServerError, "Could not look up link.")
		return
	}

	clickResult := db.Model(&Url{}).
		Where("id = ?", url.ID).
		UpdateColumn("clicks", gorm.Expr("clicks + ?", 1))
	if clickResult.Error != nil {
		writeAPIError(w, http.StatusInternalServerError, "Could not record link click.")
		return
	}
	if clickResult.RowsAffected == 0 {
		writeAPIError(w, http.StatusNotFound, "Link not found.")
		return
	}

	http.Redirect(w, r, url.URL, http.StatusFound)
}

func writeJson(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}
