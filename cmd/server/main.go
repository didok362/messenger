package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/didok362/messenger/internal/avatar"
	"github.com/didok362/messenger/internal/repository/postgres"
	"github.com/didok362/messenger/internal/service"
	transporthttp "github.com/didok362/messenger/internal/transport/http"
	"github.com/didok362/messenger/internal/transport/ws"
)

func main() {
	ctx := context.Background()

	// 1. Подключение к БД
	connStr := "postgres://didok362:qwerty67@localhost:5432/messenger_db?sslmode=disable"
	pool, err := postgres.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer pool.Close()

	jwtSecret := "super-secret-jwt-key"

	// 2. Инициализация репозиториев и сервисов
	userRepo := postgres.NewUserRepository(pool)
	chatRepo := postgres.NewChatRepository(pool)
	msgRepo := postgres.NewMessageRepository(pool)
	hub := ws.NewHub(chatRepo)
	go hub.Run()
	authService := service.NewAuthService(userRepo, jwtSecret, 24*time.Hour)
	authHandler := transporthttp.NewAuthHandler(authService)
	chatService := service.NewChatService(chatRepo, msgRepo, userRepo, hub)
	chatHandler := transporthttp.NewChatHandler(chatService, hub)
	avatarGen, err := avatar.NewGenerator("assets/avatars")
	if err != nil {
		log.Fatalf("Failed to connect to find assets: %v", err)
	}
	userHandler := transporthttp.NewUserHandler(userRepo, avatarGen)
	wsHandler := ws.NewHandler(hub, chatService, []byte(jwtSecret))

	// 3. Роутер со всеми хендлерами и секретом
	router := transporthttp.NewRouter(authHandler, chatHandler, wsHandler, userHandler, []byte(jwtSecret))

	// 4. Старт HTTP сервера
	port := ":8080"
	fmt.Printf("🚀 Сервер запущен на http://localhost%s\n", port)
	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
