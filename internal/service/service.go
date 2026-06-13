package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"shortlink/internal/storage"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
const codeLen = 10
const maxRetries = 5

type Service struct {
	storage storage.Storage
}

func New(storage storage.Storage) *Service {
	return &Service{storage: storage}
}

func (s *Service) GetShort(ctx context.Context, originalURL string) (string, error) {
	for range maxRetries {
		code := generateCode()
		err := s.storage.Save(ctx, code, originalURL)
		if err == nil {
			return code, nil
		}
		if errors.Is(err, storage.ErrURLConflict) {
			return s.storage.GetByOriginal(ctx, originalURL)
		}
		if errors.Is(err, storage.ErrCodeConflict) {
			continue
		}
		return "", fmt.Errorf("shorten: %w", err)
	}
	return "", fmt.Errorf("shorten: exceeded max retries")
}

func (s *Service) GetOriginal(ctx context.Context, code string) (string, error) {
	return s.storage.GetByCode(ctx, code)
}

func generateCode() string {
	b := make([]byte, codeLen)
	for i := range b {
		b[i] = alphabet[rand.N(len(alphabet))]
	}
	return string(b)
}
