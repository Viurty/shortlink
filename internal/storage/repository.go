package storage

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("short URL not found")
	ErrConflict = errors.New("original URL already exists") // Нужно ли возвращать ошибку или просто вернуть существующее сокращение?
)

type Storage interface {
	Save(ctx context.Context, originalURL string) (shortURL string, err error)
	Get(ctx context.Context, shortURL string) (originalURL string, err error)
}
