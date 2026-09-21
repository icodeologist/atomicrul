package api

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"

	"gorm.io/gorm"
)

var store *sessions.CookieStore

var loginPage = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Log in · AtomicURL</title>
  <style>body{font:16px system-ui,sans-serif;background:#f4f6f8;margin:0}main{max-width:420px;margin:10vh auto;padding:2rem;background:white;border:1px solid #d7dde3;border-radius:12px}label{display:block;font-weight:600;margin:1rem 0 .25rem}input,button{width:100%;box-sizing:border-box;padding:.65rem;font:inherit;border:1px solid #aeb7c1;border-radius:6px}button{margin-top:1.25rem;background:#1769e0;color:white;border:0;cursor:pointer}.error{color:#b42318}</style>
</head>
<body><main><h1>AtomicURL</h1><p>Log in to manage your links.</p>{{if .}}<p class="error">{{.}}</p>{{end}}<form method="post" action="/login"><label for="username">Username</label><input id="username" name="username" required autofocus><label for="password">Password</label><input id="password" name="password" type="password" required><button>Log in</button></form></main></body>
</html>`))

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
			writeAPIError(w, http.StatusInternalServerError, "Could not create account.")
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
			writeAPIError(w, http.StatusInternalServerError, "Could not create account.")
			return
		}

		writeJson(w, http.StatusCreated, apiSuccess{Success: "Account created successfully."})
	} else {
		writeJson(w, http.StatusMethodNotAllowed, apiError{Err: "Only post allowed"})
	}

}

func Login(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = loginPage.Execute(w, nil)
		return
	}
	if r.Method != http.MethodPost {
		writeJson(w, http.StatusMethodNotAllowed, apiError{Err: "Method not allowed."})
		return
	}

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
	if strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html") {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
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
