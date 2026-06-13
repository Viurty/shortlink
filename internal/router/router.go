package router

import (
	"net/http"
	"shortlink/internal/handler"
)

func New(h handler.API) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/shorten", h.Shorten)
	mux.HandleFunc("GET /api/expand/{short}", h.GetOriginal)
	return mux
}
