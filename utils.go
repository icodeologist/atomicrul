package main

import (
	"net/http"

	"gorm.io/gorm"
)

func FetchAllUrls(w http.ResponseWriter, r *http.Request, db *gorm.DB) []Url {
	session, _ := store.Get(r, "atomicrul")

	if session.Values["authenticated"] != true {
		writeJson(w, http.StatusUnauthorized, apiError{Err: "Please log in."})
		return nil
	}

	userId := session.Values["userid"]
	if userId == 0 {
		writeJson(w, http.StatusUnauthorized, apiError{Err: "Please log in."})
		return nil
	}

	var urls []Url
	res := db.Where("user_id=?", userId).Find(&urls)
	if res.Error != nil {
		writeJson(w, http.StatusInternalServerError, apiError{Err: res.Error.Error()})
		return nil
	}
	return urls

}
