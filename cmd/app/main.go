package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"shortlink/internal/handler"
	"shortlink/internal/router"
	"shortlink/internal/service"
	"shortlink/internal/storage"
	"shortlink/internal/storage/inmemory"
	"shortlink/internal/storage/postgres"
	"time"
)

const (
	storagePostgres = "postgresql"
	storageMemory   = "inmemory"
)

func main() {
	storageType := os.Getenv("STORAGE")
	if storageType == "" {
		log.Fatal("STORAGE не установлен")
	}

	var store storage.Storage

	switch storageType {
	case storagePostgres:
		dsn := os.Getenv("DSN")
		if dsn == "" {
			log.Fatal("DSN не установлен")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		db, err := postgres.New(ctx, dsn)
		if err != nil {
			log.Fatalf("ошибка подключения к postgres: %v", err)
		}
		defer db.Close()

		err = postgres.Migrate(dsn)
		if err != nil {
			log.Fatalf("ошибка создания таблицы: %v", err)
		}
		store = db

	case storageMemory:
		store = inmemory.New()

	default:
		log.Fatalf("неизвестный тип хранилища: %s", storageType)
	}

	svc := service.New(store)
	handler := handler.New(svc)
	router := router.New(handler)

	log.Println("сервер запущен на :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("ошибка запуска сервера: %v", err)
	}
}
