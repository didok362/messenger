package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/didok362/messenger/internal/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddleware(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing auhoriztion header", http.StatusUnauthorized)
				return
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}
			tokenString := parts[1]
			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return jwtSecret, nil
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
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))

		})

	}
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return id, ok
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 1. Оборачиваем ResponseWriter, чтобы поймать HTTP статус (200, 400, 500)
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		// 2. Пропускаем запрос дальше к нашему хэндлеру
		next.ServeHTTP(ww, r)
		// 3. Достаем шаблон пути (/api/v1/chats/{id}), чтобы не плодить миллионы метрик с UUID
		path := r.URL.Path
		if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
			path = rctx.RoutePattern()
		}
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(ww.Status())
		// 4. Записываем в метрики
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}
