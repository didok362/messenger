package ws

import (
	"errors"
	"net/http"
	"strings"

	"github.com/didok362/messenger/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Handler struct {
	hub         *Hub
	chatService *service.ChatService
	jwtSecret   []byte
}

func NewHandler(hub *Hub, chatService *service.ChatService, jwtSecret []byte) *Handler {
	return &Handler{
		hub:         hub,
		chatService: chatService,
		jwtSecret:   jwtSecret,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token")
	if strings.TrimSpace(tokenString) == "" {
		http.Error(w, "unauth", http.StatusUnauthorized)
		return
	}
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return h.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "unexpected signing method", http.StatusUnauthorized)
		return
	}
	userIDStr, ok := claims["sub"].(string)
	if !ok {
		http.Error(w, "unexpected signing method", http.StatusUnauthorized)
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "invalid user id in token", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &Client{
		hub:         h.hub,
		conn:        conn,
		send:        make(chan any, 256),
		userID:      userID,
		chatService: h.chatService,
	}
	h.hub.register <- client
	go client.writePump()
	go client.readPump()

}
