package main

import (
	"net/http"
	"time"

	"gorm.io/gorm"
)

// Get all the expired links from the user
// then remake them and update the database

func RemakeExpiredLinks(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	session, _ := store.Get(r, "atomicurl")
	if session.Values["authenticated"] != true {
		writeJson(w, http.StatusUnauthorized, apiError{Err: "Please log in."})
		return
	}
	// get user id
	id := session.Values["userid"]
	if id == 0 {
		writeJson(w, http.StatusBadRequest, apiError{Err: "Invalid user id."})
		return
	}

	var urls []Url
	res := db.Where("user_id=?", id).Find(&urls)
	if res.Error != nil {
		writeJson(w, http.StatusInternalServerError, apiError{Err: res.Error.Error()})
		return
	}

	var expiredUrls []Url
	for _, v := range urls {
		if time.Now().After(v.ExpirationTime) {
			expiredUrls = append(expiredUrls, v)
		}
	}

	for _, v := range expiredUrls {
		v.ShortLinkCreatedTime = time.Now()
		v.ExpirationTime = time.Now().Add(10 * time.Hour)
		db.Save(&v)
	}

	var reFetch []Url

	for _, v := range urls {
		if time.Now().After(v.ExpirationTime) {
			reFetch = append(reFetch, v)
		}
	}
	writeJson(w, 200, apiSuccess{Message: "All links have been reset to new expiration time."})
}
