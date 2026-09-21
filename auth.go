package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"

	"gorm.io/gorm"
)

var store *sessions.CookieStore

func ConfigureSessionStore(secret string, secure bool) error {
	if len(secret) < 32 {
		return errors.New("SECRETKEY must be at least 32 characters long")
	}

	store = sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
	return nil
}

func getSession(w http.ResponseWriter, r *http.Request) (*sessions.Session, bool) {
	if store == nil {
		writeAPIError(w, http.StatusInternalServerError, "Session service is not configured.")
		return nil, false
	}

	session, err := store.Get(r, "atomicurl")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Invalid session cookie.")
		return nil, false
	}
	return session, true
}

func Register(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method == http.MethodPost {
		// get the user request
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		if len(username) < 5 || len(password) < 8 {
			writeJson(w, http.StatusBadRequest, apiError{
				Err: "Username must be at least 5 characters and password must be at least 8 characters.",
			})
			return
		}

		// check if username is already exists
		var userNameCheck User
		db.Where("user_name=?", username).First(&userNameCheck)
		if userNameCheck.ID != 0 {
			// user already exists
			writeJson(w, http.StatusBadRequest, apiError{Err: "Username is taken. Please recheck or login if you have already registered."})
			return
		}
		//check if the email is already taken
		var emailFound User
		db.Where("email=?", email).First(&emailFound)
		if emailFound.ID != 0 {
			// user already exists
			writeJson(w, http.StatusBadRequest, apiError{Err: "Email already exists.Please login."})
			return
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			writeJson(w, http.StatusInternalServerError, apiError{Err: fmt.Sprintf("Password Hashing error : %v\n", err.Error())})
			return
		}
		//create the user
		user := User{
			UserName: username,
			Email:    email,
			Password: string(passwordHash),
		}

		res := db.Create(&user)
		if res.Error != nil {
			writeJson(w, http.StatusInternalServerError, apiError{Err: fmt.Sprintf("Error caused while creating the user %v\n", res.Error.Error())})
			return
		}

		writeJson(w, http.StatusCreated, apiSuccess{Success: "Account created successfully."})
	} else {
		writeJson(w, http.StatusMethodNotAllowed, apiError{Err: "Only post allowed"})
	}

}

func Login(w http.ResponseWriter, r *http.Request, db *gorm.DB) {

	username := r.FormValue("username")
	password := r.FormValue("password")
	// find the username from DB and check its password
	var userCheck User
	db.Where("user_name=?", username).First(&userCheck)
	if userCheck.ID == 0 {
		writeJson(w, http.StatusUnauthorized, apiError{Err: "Invalid username or password."})
		return
	}
	// check for password matching
	if err := bcrypt.CompareHashAndPassword([]byte(userCheck.Password), []byte(password)); err != nil {
		writeJson(w, http.StatusUnauthorized, apiError{Err: "Invalid username or password."})
		return
	}
	session, ok := getSession(w, r)
	if !ok {
		return
	}
	session.Values["authenticated"] = true
	session.Values["userid"] = userCheck.ID
	if err := session.Save(r, w); err != nil {
		writeJson(w, http.StatusInternalServerError, apiError{Err: "Could not start session."})
		return
	}
	writeJson(w, http.StatusOK, apiSuccess{Success: "Successfully logged in."})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session, ok := getSession(w, r)
	if !ok {
		return
	}
	session.Values = nil
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		writeJson(w, http.StatusInternalServerError, apiError{Err: "Could not end session."})
		return
	}
	writeJson(w, http.StatusOK, apiSuccess{Success: "Successfully logged out."})
}

func GreetIn(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	session, ok := getSession(w, r)
	if !ok {
		return
	}
	// get the user id
	if session.Values["authenticated"] != true {
		writeJson(w, http.StatusForbidden, apiError{Err: "You need to login in."})
		return
	}
	id := session.Values["userid"]
	var user User
	db.Where("id=?", id).First(&user)
	writeJson(w, http.StatusOK, apiSuccess{Success: fmt.Sprintf("user id : %v\n", id)})

}
