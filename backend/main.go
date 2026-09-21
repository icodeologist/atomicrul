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
	"github.com/icodeologist/atomicurl/internal/api"
	"github.com/icodeologist/atomicurl/internal/config"
	"github.com/icodeologist/atomicurl/internal/db"
	"github.com/icodeologist/atomicurl/internal/routes"
)

func main() {
	appConfig, err := config.LoadConfig(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	if err := api.ConfigureSessionStore(appConfig.SecretKey, appConfig.AppEnv == "production"); err != nil {
		log.Fatal(err)
	}

	database, err := db.SetUpDatabaseWithConfig(appConfig)
	if err != nil {
		log.Fatal(err)
	}
	if os.Getenv("ATOMICURL_SEED_DEMO") == "true" {
		if appConfig.AppEnv == "production" {
			log.Fatal("ATOMICURL_SEED_DEMO cannot be enabled in production")
		}
		if err := db.SeedDemoData(database); err != nil {
			log.Fatalf("could not seed demo data: %v", err)
		}
		log.Println("demo data ready: login with demo / demo-password")
	}
	defer func() {
		if err := db.CloseDatabase(database); err != nil {
			log.Printf("database cleanup failed: %v", err)
		}
	}()

	router := mux.NewRouter()
	routes.SetUpRoutes(router, database)
	server := &http.Server{
		Addr:              ":" + appConfig.HTTPPort,
		Handler:           router,
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
