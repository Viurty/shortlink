package router

import (
	"net/http"
	"shortlink/internal/handler"
)

func New(h *handler.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/shorten", h.Shorten)
	mux.HandleFunc("GET /api/{short}", h.GetOriginal)
	return mux
}
