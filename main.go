package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func main() {
	config, err := LoadConfig(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	if err := ConfigureSessionStore(config.SecretKey, config.AppEnv == "production"); err != nil {
		log.Fatal(err)
	}

	db, err := SetUpDbWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := CloseDatabase(db); err != nil {
			log.Printf("database cleanup failed: %v", err)
		}
	}()

	r := mux.NewRouter()
	r.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		Register(w, r, db)
	}).Methods("POST")

	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		Login(w, r, db)
	}).Methods("POST")

	rateLimiter := RateLimiterMiddleware(rate.Limit(5), 10)

	r.Handle("/create", rateLimiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleUserUrlsSumbmission(w, r, db)
	}))).Methods(http.MethodPost)
	r.HandleFunc("/links", func(w http.ResponseWriter, r *http.Request) {
		CreateLink(w, r, db)
	}).Methods(http.MethodPost)
	r.HandleFunc("/links/{id}", func(w http.ResponseWriter, r *http.Request) {
		UpdateLink(w, r, db)
	}).Methods(http.MethodPatch)
	r.HandleFunc("/links/{id}/history", func(w http.ResponseWriter, r *http.Request) {
		GetLinkHistory(w, r, db)
	}).Methods(http.MethodGet)
	r.HandleFunc("/links/{id}/versions/{versionID}/rollback", func(w http.ResponseWriter, r *http.Request) {
		RollbackLink(w, r, db)
	}).Methods(http.MethodPost)

	r.HandleFunc("/logout", Logout).Methods(http.MethodPost)
	r.HandleFunc("/greetme", func(w http.ResponseWriter, r *http.Request) {
		GreetIn(w, r, db)
	}).Methods(http.MethodGet)

	r.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		ShowUserDashBoard(w, r, db)
	}).Methods(http.MethodGet)

	r.HandleFunc("/{code}", func(w http.ResponseWriter, r *http.Request) {
		HandleRedirectionOfShortUrlToLongUrl(w, r, db)
	}).Methods(http.MethodGet)

	server := &http.Server{
		Addr:              ":" + config.HTTPPort,
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownSignals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("server listening on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server stopped: %v", err)
		}
	case <-shutdownSignals.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
		cancel()
		<-serverErrors
	}
}

func SetUpDb() (*gorm.DB, error) {
	config, err := LoadConfig(os.Getenv)
	if err != nil {
		return nil, err
	}
	return SetUpDbWithConfig(config)
}

func SetUpDbWithConfig(config AppConfig) (*gorm.DB, error) {
	databse, err := ConnectToDatabaseWithConfig(config)
	if err != nil {
		return nil, err
	}
	db := databse.DB

	if err := MigrateDatabase(db); err != nil {
		_ = CloseDatabase(db)
		return nil, err
	}
	return db, nil
}
