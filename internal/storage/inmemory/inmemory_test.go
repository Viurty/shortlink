package inmemory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"shortlink/internal/storage"
)

func TestStoreSaveAndLookup(t *testing.T) {
	store := New()
	ctx := context.Background()

	if err := store.Save(ctx, "short_code", "https://example.com"); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	code, err := store.GetByOriginal(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("GetByOriginal returned error: %v", err)
	}
	if code != "short_code" {
		t.Fatalf("unexpected code: got %q want %q", code, "short_code")
	}

	originalURL, err := store.GetByCode(ctx, "short_code")
	if err != nil {
		t.Fatalf("GetByCode returned error: %v", err)
	}
	if originalURL != "https://example.com" {
		t.Fatalf("unexpected original URL: got %q want %q", originalURL, "https://example.com")
	}
}

func TestStoreConflicts(t *testing.T) {
	store := New()
	ctx := context.Background()

	if err := store.Save(ctx, "short_code", "https://example.com"); err != nil {
		t.Fatalf("initial Save returned error: %v", err)
	}

	if err := store.Save(ctx, "another_code", "https://example.com"); !errors.Is(err, storage.ErrURLConflict) {
		t.Fatalf("expected ErrURLConflict, got %v", err)
	}

	if err := store.Save(ctx, "short_code", "https://another.example.com"); !errors.Is(err, storage.ErrCodeConflict) {
		t.Fatalf("expected ErrCodeConflict, got %v", err)
	}
}

func TestStoreNotFound(t *testing.T) {
	store := New()
	ctx := context.Background()

	if _, err := store.GetByOriginal(ctx, "https://missing.example.com"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if _, err := store.GetByCode(ctx, "missing"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCancelledContextDoesNotSave(t *testing.T) {
	store := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store.Save(ctx, "code", "https://example.com")

	if _, err := store.GetByCode(context.Background(), "code"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("cancelled Save must not persist data")
	}
}

func TestConcurrentSave(t *testing.T) {
	store := New()
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			store.Save(ctx, fmt.Sprintf("code%d", i), fmt.Sprintf("https://url%d.com", i))
		}(i)
	}
	wg.Wait()

	for i := range 100 {
		if _, err := store.GetByCode(ctx, fmt.Sprintf("code%d", i)); err != nil {
			t.Fatalf("GetByCode(%d) after concurrent save: %v", i, err)
		}
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	store := New()
	ctx := context.Background()
	store.Save(ctx, "code", "https://url.com")

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			store.Save(ctx, fmt.Sprintf("code%d", i), fmt.Sprintf("https://url%d.com", i))
		}(i)
		go func(i int) {
			defer wg.Done()
			store.GetByCode(ctx, fmt.Sprintf("code%d", i))
		}(i)
	}
	wg.Wait()
}
