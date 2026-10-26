package http

import (
	"net/http"

	"github.com/didok362/messenger/internal/transport/ws"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(authHandler *AuthHandler, chatHandler *ChatHandler, wsHandler *ws.Handler, userHandler *UserHandler, jwtSecret []byte) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/index.html")
	})

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			//WebSocket
			r.Get("/ws", wsHandler.ServeWS)

			// Публичные маршруты
			r.Route("/auth", func(r chi.Router) {
				r.Post("/register", authHandler.Register)
				r.Post("/login", authHandler.Login)
			})
			r.Get("/users/{id}/avatar", userHandler.GetAvatar)
			// Защищённые маршруты
			r.Group(func(r chi.Router) {
				r.Use(AuthMiddleware(jwtSecret))

				r.Route("/users", func(r chi.Router) {
					r.Get("/by-username", userHandler.GetByUsername)
				})
				r.Route("/chats", func(r chi.Router) {
					r.Get("/", chatHandler.GetUserChats)
					r.Post("/direct", chatHandler.CreateDirect)
					r.Post("/group", chatHandler.CreateGroupChat)

					r.Get("/{id}/messages", chatHandler.GetChatMessages)
					r.Post("/{id}/messages", chatHandler.SendMessage)
					r.Post("/{id}/read", chatHandler.MarkAsRead)
				})
			})
		})
	})

	return r
}
