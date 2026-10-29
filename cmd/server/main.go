package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/didok362/messenger/internal/avatar"
	"github.com/didok362/messenger/internal/config"
	"github.com/didok362/messenger/internal/events"
	"github.com/didok362/messenger/internal/repository/postgres"
	"github.com/didok362/messenger/internal/service"
	transporthttp "github.com/didok362/messenger/internal/transport/http"
	"github.com/didok362/messenger/internal/transport/ws"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	kafkaProducer := events.NewProducer(cfg.KafkaBrokersList(), cfg.KafkaTopicMessages)
	defer kafkaProducer.Close()

	// 1. Подключение к БД
	pool, err := postgres.New(ctx, cfg.DB.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer pool.Close()

	// 2. Инициализация репозиториев и сервисов
	userRepo := postgres.NewUserRepository(pool)
	chatRepo := postgres.NewChatRepository(pool)
	msgRepo := postgres.NewMessageRepository(pool)

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr(),
	})
	defer rdb.Close()

	hub := ws.NewHub(rdb)
	go hub.Run()

	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
	authHandler := transporthttp.NewAuthHandler(authService)
	chatService := service.NewChatService(chatRepo, msgRepo, userRepo, hub, kafkaProducer)
	chatHandler := transporthttp.NewChatHandler(chatService, hub)
	avatarGen, err := avatar.NewGenerator(cfg.AssetsDir)

	events.StartWSDeliveryConsumer(ctx, cfg.KafkaBrokersList(), cfg.KafkaTopicMessages, hub)
	events.StartDBArchiverConsumer(ctx, cfg.KafkaBrokersList(), cfg.KafkaTopicMessages, msgRepo)

	if err != nil {
		log.Fatalf("Failed to connect to find assets: %v", err)
	}
	userHandler := transporthttp.NewUserHandler(userRepo, avatarGen)
	wsHandler := ws.NewHandler(hub, chatService, []byte(cfg.JWTSecret))

	// 3. Роутер со всеми хендлерами и секретом
	router := transporthttp.NewRouter(authHandler, chatHandler, wsHandler, userHandler, []byte(cfg.JWTSecret))

	server := &http.Server{
		Addr:    ":" + strings.TrimPrefix(cfg.HTTPPort, ":"),
		Handler: router,
	}

	go func() {
		fmt.Printf("🚀 Сервер запущен на http://localhost%s\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка сервера: %v", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("\n🛑 Получен сигнал остановки, мягко выключаем сервер...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при остановке сервера: %v", err)
	}

	fmt.Println("👋 Сервер успешно остановлен")

}
