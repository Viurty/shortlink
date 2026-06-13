package storage

import (
	"context"
	"errors"
)

var (
	ErrNotFound     = errors.New("short code not found")
	ErrURLConflict  = errors.New("original URL already exists")
	ErrCodeConflict = errors.New("short code already exists")
)

type Storage interface {
	Save(ctx context.Context, code, originalURL string) error
	GetByOriginal(ctx context.Context, originalURL string) (string, error)
	GetByCode(ctx context.Context, shortCode string) (string, error)
}
