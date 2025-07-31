package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"

	"gorm.io/gorm"
)

var retry = 0
var store = sessions.NewCookieStore([]byte(os.Getenv("SECRETKEY")))

func Register(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method == http.MethodPost {
		// get the user request
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		if len(username) <= 5 || len(password) < 8 {
			writeJson(w, http.StatusBadRequest, apiError{
				Err: "Username should atleast be 5 characters long. Passwords must be 8 or highe.",
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

		writeJson(w, http.StatusOK, apiSuccess{Message: fmt.Sprintf("User created %v\n", user)})
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
		writeJson(w, http.StatusNotFound, apiError{Err: fmt.Sprintf("User does not exists : %v\n", username)})
		return
	}
	// check for password matching
	if err := bcrypt.CompareHashAndPassword([]byte(userCheck.Password), []byte(password)); err != nil {
		writeJson(w, http.StatusNotFound, apiError{Err: fmt.Sprintf("haa Try again you dum fuk")})
		retry++
		if retry == 3 {
			writeJson(w, http.StatusNotFound, apiError{Err: fmt.Sprintf("Hahhahahha get banned. You mom's third cat has a better memory. You re ta hard")})
			return
		}
		return
	}
	retry = 0
	session, _ := store.Get(r, "atomicurl")
	session.Values["authenticated"] = true
	session.Values["userid"] = userCheck.ID
	session.Save(r, w)
	writeJson(w, http.StatusOK, apiSuccess{Message: "Successfully logged in."})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "atomicurl")
	session.Values["authenticated"] = false
	session.Save(r, w)
	writeJson(w, http.StatusOK, apiSuccess{Message: "Successfully logged out."})
}

func GreetIn(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	session, _ := store.Get(r, "atomicurl")
	// get the user id
	if session.Values["authenticated"] != true {
		writeJson(w, http.StatusForbidden, apiError{Err: "You need to login in."})
		return
	}
	id := session.Values["userid"]
	var user User
	db.Where("id=?", id).First(&user)
	fmt.Println(user)
	writeJson(w, http.StatusOK, apiSuccess{Message: fmt.Sprintf("user id : %v\n", id)})

}
