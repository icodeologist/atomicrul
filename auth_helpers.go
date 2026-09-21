package main

import (
	"errors"
	"net/http"

	"gorm.io/gorm"
)

func requireAuthenticatedUser(w http.ResponseWriter, r *http.Request) (uint, bool) {
	session, ok := getSession(w, r)
	if !ok {
		return 0, false
	}
	if session.Values["authenticated"] != true {
		writeAPIError(w, http.StatusUnauthorized, "Please log in.")
		return 0, false
	}

	userID, ok := sessionUserID(session.Values["userid"])
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Please log in.")
		return 0, false
	}
	return userID, true
}

func findOwnedLink(db *gorm.DB, userID, linkID uint) (Link, error) {
	var link Link
	result := db.Where("id = ? AND user_id = ?", linkID, userID).First(&link)
	return link, result.Error
}

func writeAPIError(w http.ResponseWriter, statusCode int, message string) {
	writeJson(w, statusCode, apiError{Err: message})
}

func sessionUserID(value any) (uint, bool) {
	switch id := value.(type) {
	case uint:
		return id, id != 0
	case uint64:
		return uint(id), id != 0 && uint64(uint(id)) == id
	case int:
		return uint(id), id > 0
	case int64:
		return uint(id), id > 0 && uint64(uint(id)) == uint64(id)
	default:
		return 0, false
	}
}

func isLinkNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
