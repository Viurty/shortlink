package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"shortlink/internal/service"
	"shortlink/internal/storage"
)

type fakeStore struct {
	codeToOriginal map[string]string
	originalToCode map[string]string
	saveErr        error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		codeToOriginal: make(map[string]string),
		originalToCode: make(map[string]string),
	}
}

func (f *fakeStore) Save(ctx context.Context, code, originalURL string) error {
	if f.saveErr != nil {
		err := f.saveErr
		f.saveErr = nil
		return err
	}
	if _, ok := f.originalToCode[originalURL]; ok {
		return storage.ErrURLConflict
	}
	if _, ok := f.codeToOriginal[code]; ok {
		return storage.ErrCodeConflict
	}
	f.codeToOriginal[code] = originalURL
	f.originalToCode[originalURL] = code
	return nil
}

func (f *fakeStore) GetByOriginal(ctx context.Context, originalURL string) (string, error) {
	if c, ok := f.originalToCode[originalURL]; ok {
		return c, nil
	}
	return "", storage.ErrNotFound
}

func (f *fakeStore) GetByCode(ctx context.Context, code string) (string, error) {
	if u, ok := f.codeToOriginal[code]; ok {
		return u, nil
	}
	return "", storage.ErrNotFound
}

func TestShortenSuccess(t *testing.T) {
	store := newFakeStore()
	svc := service.New(store)
	h := New(svc)

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Shorten(rr, req)

	if rr.Code != 201 {
		t.Fatalf("status = %d; want 201", rr.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	code := body["code"]
	if code == "" {
		t.Fatal("expected non-empty code, got empty")
	}
	if len(code) != 10 {
		t.Fatalf("code length = %d; want 10", len(code))
	}
}

func TestShortenDuplicateURL(t *testing.T) {
	store := newFakeStore()
	svc := service.New(store)
	h := New(svc)

	doPost := func() string {
		req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"url":"https://example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.Shorten(rr, req)
		if rr.Code != 201 {
			t.Fatalf("status = %d; want 201", rr.Code)
		}
		var body map[string]string
		json.NewDecoder(rr.Body).Decode(&body)
		return body["code"]
	}

	code1 := doPost()
	code2 := doPost()

	if code1 != code2 {
		t.Fatalf("duplicate URL must return the same code: got %q and %q", code1, code2)
	}
}

func TestShortenEmptyURL(t *testing.T) {
	store := newFakeStore()
	svc := service.New(store)
	h := New(svc)

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"url":""}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Shorten(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status = %d; want 400", rr.Code)
	}
}

func TestShortenWhitespaceURL(t *testing.T) {
	store := newFakeStore()
	svc := service.New(store)
	h := New(svc)

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"url":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Shorten(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status = %d; want 400", rr.Code)
	}
}

func TestShortenBadJSON(t *testing.T) {
	store := newFakeStore()
	svc := service.New(store)
	h := New(svc)

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{bad json}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Shorten(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status = %d; want 400", rr.Code)
	}
}

func TestShortenUnknownField(t *testing.T) {
	store := newFakeStore()
	svc := service.New(store)
	h := New(svc)

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"url":"https://example.com","extra":"field"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Shorten(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status = %d; want 400", rr.Code)
	}
}

func TestShortenResponseIsJSON(t *testing.T) {
	store := newFakeStore()
	svc := service.New(store)
	h := New(svc)

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Shorten(rr, req)

	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type = %q; want application/json", ct)
	}
}

func TestGetOriginalSuccess(t *testing.T) {
	store := newFakeStore()
	store.codeToOriginal["abc1234567"] = "https://example.com"
	store.originalToCode["https://example.com"] = "abc1234567"
	svc := service.New(store)
	h := New(svc)

	req := httptest.NewRequest("GET", "/abc1234567", nil)
	req.SetPathValue("short", "abc1234567")
	rr := httptest.NewRecorder()

	h.GetOriginal(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status = %d; want 200", rr.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got, want := body["url"], "https://example.com"; got != want {
		t.Fatalf("url = %q; want %q", got, want)
	}
}

func TestGetOriginalNotFound(t *testing.T) {
	store := newFakeStore()
	svc := service.New(store)
	h := New(svc)

	req := httptest.NewRequest("GET", "/missing123", nil)
	req.SetPathValue("short", "missing123")
	rr := httptest.NewRecorder()

	h.GetOriginal(rr, req)

	if rr.Code != 404 {
		t.Fatalf("status = %d; want 404", rr.Code)
	}
}

func TestGetOriginalEmptyCode(t *testing.T) {
	store := newFakeStore()
	svc := service.New(store)
	h := New(svc)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	h.GetOriginal(rr, req)

	if rr.Code != 404 {
		t.Fatalf("status = %d; want 404", rr.Code)
	}
}

func TestGetOriginalResponseIsJSON(t *testing.T) {
	store := newFakeStore()
	store.codeToOriginal["abc1234567"] = "https://example.com"
	store.originalToCode["https://example.com"] = "abc1234567"
	svc := service.New(store)
	h := New(svc)

	req := httptest.NewRequest("GET", "/abc1234567", nil)
	req.SetPathValue("short", "abc1234567")
	rr := httptest.NewRecorder()

	h.GetOriginal(rr, req)

	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type = %q; want application/json", ct)
	}
}
