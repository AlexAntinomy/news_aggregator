package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Ключ для хранения ID запроса в контексте
type RequestIDKey string

const (
	// RequestIDHeader is the header name for the request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDContextKey is the context key for the request ID
	RequestIDContextKey RequestIDKey = "request_id"
)

// responseWriter - обертка для http.ResponseWriter для перехвата статус-кода
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader перехватывает статус-код
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// MetricsMiddleware создает middleware для сбора метрик HTTP-запросов
func MetricsMiddleware(next http.Handler, duration *prometheus.HistogramVec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем ResponseWriter для перехвата статус-кода
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Выполняем следующий обработчик
		next.ServeHTTP(rw, r)

		// Записываем метрики
		duration.WithLabelValues(
			r.Method,
			r.URL.Path,
			string(rw.statusCode),
		).Observe(time.Since(start).Seconds())
	})
}

// LoggingMiddleware создает middleware для логирования HTTP-запросов
func LoggingMiddleware(next http.Handler, logger *logrus.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем ResponseWriter для перехвата статус-кода
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Выполняем следующий обработчик
		next.ServeHTTP(rw, r)

		// Логируем информацию о запросе
		duration := time.Since(start)
		logger.WithFields(logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     rw.statusCode,
			"duration":   duration,
			"user_agent": r.UserAgent(),
			"remote_ip":  r.RemoteAddr,
		}).Info("HTTP request")
	})
}

// RequestIDMiddleware создает middleware для добавления request ID
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Генерируем уникальный ID запроса
		requestID := generateRequestID()

		// Добавляем ID в заголовок ответа
		w.Header().Set("X-Request-ID", requestID)

		// Добавляем ID в контекст запроса
		ctx := r.Context()
		ctx = context.WithValue(ctx, "request_id", requestID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// generateRequestID генерирует уникальный ID запроса
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "error-generating-id"
	}
	return hex.EncodeToString(b)
}
