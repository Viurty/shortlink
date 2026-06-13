package inmemory

import (
	"context"
	"shortlink/internal/storage"
	"sync"
)

type Store struct {
	mu             sync.RWMutex
	codeToOriginal map[string]string
	originalToCode map[string]string
}

func New() *Store {
	return &Store{
		codeToOriginal: make(map[string]string),
		originalToCode: make(map[string]string),
	}
}

func (s *Store) Save(ctx context.Context, code, originalURL string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.originalToCode[originalURL]; ok {
		return storage.ErrURLConflict
	}
	if _, ok := s.codeToOriginal[code]; ok {
		return storage.ErrCodeConflict
	}

	s.codeToOriginal[code] = originalURL
	s.originalToCode[originalURL] = code
	return nil
}

func (s *Store) GetByOriginal(ctx context.Context, originalURL string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	code, ok := s.originalToCode[originalURL]
	if !ok {
		return "", storage.ErrNotFound
	}
	return code, nil
}

func (s *Store) GetByCode(ctx context.Context, code string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	originalURL, ok := s.codeToOriginal[code]
	if !ok {
		return "", storage.ErrNotFound
	}
	return originalURL, nil

}
