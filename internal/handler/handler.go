package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"shortlink/internal/storage"
	"strings"
)

type Shortener interface {
	GetShort(ctx context.Context, originalURL string) (string, error)
	GetOriginal(ctx context.Context, code string) (string, error)
}

type API interface {
	Shorten(w http.ResponseWriter, r *http.Request)
	GetOriginal(w http.ResponseWriter, r *http.Request)
}

type Handler struct {
	svc Shortener
}

func New(svc Shortener) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		URL string `json:"url"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	originalURL := strings.TrimSpace(req.URL)
	if originalURL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	parsedURL, err := url.ParseRequestURI(originalURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}
	code, err := h.svc.GetShort(ctx, originalURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"code": code})
}

func (h *Handler) GetOriginal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := r.PathValue("short")
	originalURL, err := h.svc.GetOriginal(ctx, code)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "short code not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"url": originalURL})
}

func writeError(w http.ResponseWriter, status int, errMessage string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": errMessage})
}
