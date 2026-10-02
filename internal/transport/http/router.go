package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(authHandler *AuthHandler, chatHandler *ChatHandler, jwtSecret []byte) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			// Публичные маршруты
			r.Route("/auth", func(r chi.Router) {
				r.Post("/register", authHandler.Register)
				r.Post("/login", authHandler.Login)
			})
			// Защищённые маршруты
			r.Group(func(r chi.Router) {
				r.Use(AuthMiddleware(jwtSecret))

				r.Route("/chats", func(r chi.Router) {
					r.Get("/", chatHandler.GetUserChats)
					r.Post("/direct", chatHandler.CreateDirect)

					r.Get("/{id}/messages", chatHandler.GetChatMessages)
					r.Post("/{id}/messages", chatHandler.SendMessage)
				})
			})
		})
	})

	return r
}
