package service

import (
	"context"
	"errors"
	"testing"

	"shortlink/internal/storage"
)

type fakeStorage struct {
	saveErrors     []error
	saveCalls      int
	savedCodes     []string
	savedURLs      []string
	originalByURL  map[string]string
	originalByCode map[string]string
}

func (f *fakeStorage) Save(ctx context.Context, code, originalURL string) error {
	f.saveCalls++
	f.savedCodes = append(f.savedCodes, code)
	f.savedURLs = append(f.savedURLs, originalURL)
	if len(f.saveErrors) == 0 {
		return nil
	}
	err := f.saveErrors[0]
	f.saveErrors = f.saveErrors[1:]
	return err
}

func (f *fakeStorage) GetByOriginal(ctx context.Context, originalURL string) (string, error) {
	if code, ok := f.originalByURL[originalURL]; ok {
		return code, nil
	}
	return "", storage.ErrNotFound
}

func (f *fakeStorage) GetByCode(ctx context.Context, shortCode string) (string, error) {
	if originalURL, ok := f.originalByCode[shortCode]; ok {
		return originalURL, nil
	}
	return "", storage.ErrNotFound
}

func TestGenerateCode(t *testing.T) {
	allowed := map[rune]struct{}{}
	for _, r := range alphabet {
		allowed[r] = struct{}{}
	}

	for range 100 {
		code := generateCode()
		if len([]rune(code)) != codeLen {
			t.Fatalf("unexpected code length: got %d want %d", len([]rune(code)), codeLen)
		}
		for _, r := range code {
			if _, ok := allowed[r]; !ok {
				t.Fatalf("unexpected character %q in code %q", r, code)
			}
		}
	}
}

func TestGetShortCreatesNewCode(t *testing.T) {
	store := &fakeStorage{}
	svc := New(store)

	code, err := svc.GetShort(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("GetShort returned error: %v", err)
	}
	if len([]rune(code)) != codeLen {
		t.Fatalf("unexpected code length: got %d want %d", len([]rune(code)), codeLen)
	}
	if store.saveCalls != 1 {
		t.Fatalf("unexpected save calls: got %d want 1", store.saveCalls)
	}
}

func TestGetShortRetriesOnCodeConflict(t *testing.T) {
	store := &fakeStorage{saveErrors: []error{storage.ErrCodeConflict, nil}}
	svc := New(store)

	code, err := svc.GetShort(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("GetShort returned error: %v", err)
	}
	if code == "" {
		t.Fatal("expected non-empty code")
	}
	if store.saveCalls != 2 {
		t.Fatalf("unexpected save calls: got %d want 2", store.saveCalls)
	}
	if len(store.savedCodes) == 2 && store.savedCodes[0] == store.savedCodes[1] {
		t.Fatal("retry must generate a new code, not repeat the same one")
	}
}

func TestGetShortFailsAfterMaxRetries(t *testing.T) {
	errs := make([]error, maxRetries+1)
	for i := range errs {
		errs[i] = storage.ErrCodeConflict
	}
	store := &fakeStorage{saveErrors: errs}
	svc := New(store)

	_, err := svc.GetShort(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}
}

func TestGetShortReturnsExistingCodeOnURLConflict(t *testing.T) {
	originalURL := "https://example.com"
	store := &fakeStorage{
		saveErrors:    []error{storage.ErrURLConflict},
		originalByURL: map[string]string{originalURL: "existingCode"},
	}
	svc := New(store)

	code, err := svc.GetShort(context.Background(), originalURL)
	if err != nil {
		t.Fatalf("GetShort returned error: %v", err)
	}
	if code != "existingCode" {
		t.Fatalf("unexpected code: got %q want %q", code, "existingCode")
	}
	if store.saveCalls != 1 {
		t.Fatalf("unexpected save calls: got %d want 1", store.saveCalls)
	}
}

func TestGetShortURLConflictButGetByOriginalFails(t *testing.T) {
	store := &fakeStorage{
		saveErrors:    []error{storage.ErrURLConflict},
		originalByURL: map[string]string{},
	}
	svc := New(store)

	_, err := svc.GetShort(context.Background(), "https://example.com")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetShortPropagatesUnknownError(t *testing.T) {
	dbErr := errors.New("connection refused")
	store := &fakeStorage{saveErrors: []error{dbErr}}
	svc := New(store)

	_, err := svc.GetShort(context.Background(), "https://example.com")
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped dbErr, got %v", err)
	}
}

func TestGetOriginalPassesThroughStorage(t *testing.T) {
	store := &fakeStorage{
		originalByCode: map[string]string{"abc": "https://example.com"},
	}
	svc := New(store)

	originalURL, err := svc.GetOriginal(context.Background(), "abc")
	if err != nil {
		t.Fatalf("GetOriginal returned error: %v", err)
	}
	if originalURL != "https://example.com" {
		t.Fatalf("unexpected original URL: got %q", originalURL)
	}
}

func TestGetOriginalReturnsNotFound(t *testing.T) {
	svc := New(&fakeStorage{})

	_, err := svc.GetOriginal(context.Background(), "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
