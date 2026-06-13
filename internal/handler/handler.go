package handler

import (
	"encoding/json"
	"net/http"
	"shortlink/internal/service"
	"strings"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
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
	if strings.TrimSpace(req.URL) == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	code, err := h.svc.GetShort(ctx, req.URL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"code": code})
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := r.PathValue("short")
	url, err := h.svc.GetOriginal(ctx, code)
	if err != nil {
		writeError(w, http.StatusNotFound, "short code not found")
		return
	}

	// http.Redirect(w, r, url, http.StatusFound)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

func writeError(w http.ResponseWriter, status int, errMessage string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": errMessage})
}
