package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 1. Счётчик HTTP-запросов (только растёт)
	// Labels (ярлыки): method (GET/POST), path (/api/v1/chats), status (200, 401, 500)
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"method", "path", "status"},
	)
	// 2. Гистограмма задержки HTTP-запросов (в секундах)
	// Buckets: "корзины" времени (1мс, 5мс, 10мс, 25мс, 50мс... до 2.5 сек)
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
		},
		[]string{"method", "path"},
	)
	// 3. Спидометр активных WebSocket-клиентов (растёт и падает)
	WebSocketActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_active_connections",
			Help: "Number of active WebSocket connections currently open",
		},
	)
	// 4. Время рассылки сообщений через Hub (микросекунды)
	WebSocketBroadcastDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "websocket_broadcast_duration_seconds",
			Help:    "Duration of WebSocket message distribution in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
		},
	)
)
