package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/didok362/messenger/internal/avatar"
	"github.com/didok362/messenger/internal/domain"
	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	userRepo  domain.UserRepository
	avatarGen *avatar.Generator
}

func NewUserHandler(userRepo domain.UserRepository, avatarGen *avatar.Generator) *UserHandler {
	return &UserHandler{
		userRepo:  userRepo,
		avatarGen: avatarGen,
	}
}

func (h *UserHandler) GetByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username query param is required", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *UserHandler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "id query param is required", http.StatusBadRequest)
		return
	}
	imgBytes, err := h.avatarGen.Generate(idStr)
	if err != nil {
		http.Error(w, "faield to create avatar", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(imgBytes)
}
