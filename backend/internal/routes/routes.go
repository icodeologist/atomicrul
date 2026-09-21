package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/icodeologist/atomicurl/internal/api"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// SetUpRoutes registers the public API, authenticated link management API,
// browser dashboard, and compatibility endpoints on the supplied router.
func SetUpRoutes(router *mux.Router, database *gorm.DB) {
	router.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		api.Register(w, r, database)
	}).Methods(http.MethodPost)
	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		api.Login(w, r, database)
	}).Methods(http.MethodGet, http.MethodPost)

	rateLimiter := api.RateLimiterMiddleware(rate.Limit(5), 10)
	router.Handle("/create", rateLimiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.HandleUserUrlsSubmission(w, r, database)
	}))).Methods(http.MethodPost)

	router.HandleFunc("/links", func(w http.ResponseWriter, r *http.Request) {
		api.CreateLink(w, r, database)
	}).Methods(http.MethodPost)
	router.HandleFunc("/links/{id}", func(w http.ResponseWriter, r *http.Request) {
		api.UpdateLink(w, r, database)
	}).Methods(http.MethodPatch)
	router.HandleFunc("/links/{id}/history", func(w http.ResponseWriter, r *http.Request) {
		api.GetLinkHistory(w, r, database)
	}).Methods(http.MethodGet)
	router.HandleFunc("/links/{id}/versions/{versionID}/rollback", func(w http.ResponseWriter, r *http.Request) {
		api.RollbackLink(w, r, database)
	}).Methods(http.MethodPost)

	router.HandleFunc("/logout", api.Logout).Methods(http.MethodPost)
	router.HandleFunc("/greetme", func(w http.ResponseWriter, r *http.Request) {
		api.GreetIn(w, r, database)
	}).Methods(http.MethodGet)
	router.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		api.ShowUserDashBoard(w, r, database)
	}).Methods(http.MethodGet)
	router.HandleFunc("/{code}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleRedirectionOfShortUrlToLongUrl(w, r, database)
	}).Methods(http.MethodGet)
}
