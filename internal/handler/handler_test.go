package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"shortlink/internal/storage"
)

type fakeShortener struct {
	shortCode   string
	shortErr    error
	originalURL string
	originalErr error
	lastURL     string
	lastCode    string
}

func (f *fakeShortener) GetShort(ctx context.Context, originalURL string) (string, error) {
	f.lastURL = originalURL
	return f.shortCode, f.shortErr
}

func (f *fakeShortener) GetOriginal(ctx context.Context, code string) (string, error) {
	f.lastCode = code
	return f.originalURL, f.originalErr
}

func TestShortenSuccess(t *testing.T) {
	svc := &fakeShortener{shortCode: "abc1234567"}
	h := New(svc)

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"url":"  https://example.com  "}`))
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
	if got := body["code"]; got != "abc1234567" {
		t.Fatalf("code = %q; want %q", got, "abc1234567")
	}
	if got := svc.lastURL; got != "https://example.com" {
		t.Fatalf("last URL = %q; want %q", got, "https://example.com")
	}
}

func TestShortenWhitespaceURL(t *testing.T) {
	h := New(&fakeShortener{})

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"url":"\u2003"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Shorten(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status = %d; want 400", rr.Code)
	}
}

func TestShortenInvalidURL(t *testing.T) {
	h := New(&fakeShortener{})

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"url":"not-a-url"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Shorten(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status = %d; want 400", rr.Code)
	}
}

func TestShortenUnknownField(t *testing.T) {
	h := New(&fakeShortener{})

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"url":"https://example.com","extra":"field"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Shorten(rr, req)

	if rr.Code != 400 {
		t.Fatalf("status = %d; want 400", rr.Code)
	}
}

func TestShortenResponseIsJSON(t *testing.T) {
	h := New(&fakeShortener{shortCode: "abc1234567"})

	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Shorten(rr, req)

	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type = %q; want application/json", ct)
	}
}

func TestGetOriginalSuccess(t *testing.T) {
	svc := &fakeShortener{originalURL: "https://example.com"}
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
	if got := svc.lastCode; got != "abc1234567" {
		t.Fatalf("last code = %q; want %q", got, "abc1234567")
	}
}

func TestGetOriginalNotFound(t *testing.T) {
	svc := &fakeShortener{originalErr: storage.ErrNotFound}
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
	svc := &fakeShortener{originalErr: storage.ErrNotFound}
	h := New(svc)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	h.GetOriginal(rr, req)

	if rr.Code != 404 {
		t.Fatalf("status = %d; want 404", rr.Code)
	}
}

func TestGetOriginalResponseIsJSON(t *testing.T) {
	h := New(&fakeShortener{originalURL: "https://example.com"})

	req := httptest.NewRequest("GET", "/abc1234567", nil)
	req.SetPathValue("short", "abc1234567")
	rr := httptest.NewRecorder()

	h.GetOriginal(rr, req)

	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type = %q; want application/json", ct)
	}
}

func TestGetOriginalInternalError(t *testing.T) {
	h := New(&fakeShortener{originalErr: errors.New("db timeout")})

	req := httptest.NewRequest("GET", "/abc1234567", nil)
	req.SetPathValue("short", "abc1234567")
	rr := httptest.NewRecorder()

	h.GetOriginal(rr, req)

	if rr.Code != 500 {
		t.Fatalf("status = %d; want 500", rr.Code)
	}
}
