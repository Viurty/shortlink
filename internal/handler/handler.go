package handler

import (
	"net/http"
	"shortlink/internal/service"
)

type Handler struct {
	svc service.Service
}

func New(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) HandleGetURL(w http.ResponseWriter, r *http.Request) {
}

func (h *Handler) HandleSaveURL(w http.ResponseWriter, r *http.Request) {
}
