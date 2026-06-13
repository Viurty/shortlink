package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"shortlink/internal/handler"
	"shortlink/internal/router"
	"shortlink/internal/service"
	"shortlink/internal/storage"
	"shortlink/internal/storage/inmemory"
	"shortlink/internal/storage/postgres"
	"syscall"
	"time"
)

const (
	storagePostgres = "postgresql"
	storageMemory   = "inmemory"
)

func main() {
	storageType := os.Getenv("STORAGE")
	if storageType == "" {
		log.Fatal("STORAGE is not set")
	}

	var store storage.Storage

	switch storageType {
	case storagePostgres:
		dsn := os.Getenv("DSN")
		if dsn == "" {
			log.Fatal("DSN is not set")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		db, err := postgres.New(ctx, dsn)
		if err != nil {
			log.Fatalf("postgres connection failed: %v", err)
		}
		defer db.Close()

		err = postgres.Migrate(dsn)
		if err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		store = db

	case storageMemory:
		store = inmemory.New()

	default:
		log.Fatalf("unknown storage type: %s", storageType)
	}

	s := service.New(store)
	h := handler.New(s)
	r := router.New(h)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("server is running on :8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server start failed: %v", err)
		}
	}()

	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
}
