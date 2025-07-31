package main

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// when user with seesion visits /dashborad
// it should showall the urls he created, then when urls expired or about to expires and stuffs
// lets start a basic one for this then can handle complicated stuffs

// so how to start
//
//first figure out the session thing
//then when user is logged in and visits /dashboard it should send the json response of url data
//may be cvreate a custom url model for this
//	url
//	shorturl
//	created
//	clicks
//	expired or out
//	when expires

type ShowUserUrldata struct {
	LongUrl              string
	ShortUrl             string
	Clicks               int
	TimeBeforeExpiration string
}

func ShowUserDashBoard(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	session, _ := store.Get(r, "atomicurl")
	//check if the user is authenticated
	if session.Values["authenticated"] != true {
		writeJson(w, http.StatusUnauthorized, apiError{Err: "Please log in to view your dashboard."})
		return
	}

	// Get the user id
	id := session.Values["userid"]
	fmt.Println("ID   _______ ", id)
	if id == 0 {
		writeJson(w, http.StatusBadRequest, apiError{Err: "Id was 0"})
		return
	}
	// Fetch all the urls user created even expired one
	var urls []Url
	res := db.Where("user_id=?", id).Find(&urls)
	if res.Error != nil {
		writeJson(w, http.StatusInternalServerError, apiError{Err: res.Error.Error()})
		return
	}
	var UrlsData []ShowUserUrldata
	for _, v := range urls {
		urldata := ShowUserUrldata{
			LongUrl:              v.URL,
			ShortUrl:             v.ShortLink,
			Clicks:               v.Clicks,
			TimeBeforeExpiration: fmt.Sprint(v.ExpirationTime.Format("15:04:05")),
		}
		// Checking if the link has already expired
		// if expired we simply update the table with Expired data
		if time.Now().After(v.ExpirationTime) {
			urldata.TimeBeforeExpiration = "Link already Expired"
		} else {
			urldata.TimeBeforeExpiration = fmt.Sprint(v.ExpirationTime.Format("15:04:05"))
		}
		UrlsData = append(UrlsData, urldata)
	}
	writeJson(w, http.StatusOK, apiSuccess{Message: UrlsData})
}
