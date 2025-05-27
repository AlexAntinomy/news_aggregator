package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// NewsShortDetailed - краткая информация о новости
type NewsShortDetailed struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Date    string `json:"date"`
}

// NewsFullDetailed - полная информация о новости с комментариями
type NewsFullDetailed struct {
	ID       int       `json:"id"`
	Title    string    `json:"title"`
	Content  string    `json:"content"`
	Date     string    `json:"date"`
	Source   string    `json:"source"`
	Comments []Comment `json:"comments"`
}

// Comment - комментарий к новости
type Comment struct {
	ID        int    `json:"id"`
	NewsID    int    `json:"news_id"`
	ParentID  *int   `json:"parent_id,omitempty"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

var (
	newsServiceURL       = os.Getenv("NEWS_SERVICE_URL")
	commentsServiceURL   = os.Getenv("COMMENTS_SERVICE_URL")
	censorshipServiceURL = os.Getenv("CENSORSHIP_SERVICE_URL")
)

func main() {
	// Настройка логгера
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	if newsServiceURL == "" {
		newsServiceURL = "http://localhost:8082"
	}
	if commentsServiceURL == "" {
		commentsServiceURL = "http://localhost:8081"
	}
	if censorshipServiceURL == "" {
		censorshipServiceURL = "http://localhost:8083"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleWelcome)
	mux.HandleFunc("/api/news", handleNewsList)
	mux.HandleFunc("/api/news/", handleNewsDetail)
	mux.HandleFunc("/api/comments", handleAddComment)

	// Подключаем middleware
	handler := RequestIDMiddleware(mux)
	handler = LoggingMiddleware(handler)

	log.Println("Запуск API Gateway на порту :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

// RequestIDMiddleware добавляет ID запроса в контекст
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware логирует информацию о каждом запросе
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := r.Header.Get("X-Request-ID")

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		logrus.WithFields(logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     rw.statusCode,
			"duration":   duration,
			"request_id": requestID,
			"remote_ip":  r.RemoteAddr,
		}).Info("Обработан запрос")
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// handleWelcome - обработчик главной страницы
func handleWelcome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
		<!DOCTYPE html>
		<html lang="ru">
		<head><meta charset="UTF-8"><title>API Gateway</title></head>
		<body>
			<h1>Добро пожаловать в API Gateway агрегатора новостей</h1>
			<ul>
				<li><a href="/api/news">/api/news</a> — Список новостей</li>
				<li><a href="/api/news/1">/api/news/&lt;id&gt;</a> — Детали новости (замените &lt;id&gt;)</li>
				<li><a href="/api/comments">/api/comments</a> — Добавление комментария (POST)</li>
			</ul>
		</body>
		</html>
	`))
}

// handleNewsList возвращает список новостей
func handleNewsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Пересылаем запрос в сервис новостей со всеми параметрами
	resp, err := http.Get(newsServiceURL + "/api/news?" + r.URL.RawQuery)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Ошибка получения новостей", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Копируем заголовки ответа
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	// Копируем тело ответа
	io.Copy(w, resp.Body)
}

// handleNewsDetail возвращает детальную информацию о конкретной новости
func handleNewsDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID новости из URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Неверный ID новости", http.StatusBadRequest)
		return
	}

	newsID := parts[3]

	// Получаем детали новости
	newsResp, err := http.Get(newsServiceURL + "/api/news/" + newsID)
	if err != nil {
		log.Printf("Ошибка получения новости: %v", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Ошибка получения новости", http.StatusInternalServerError)
		return
	}
	defer newsResp.Body.Close()

	if newsResp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Новость не найдена", http.StatusNotFound)
		return
	}

	// Получаем комментарии
	commentsResp, err := http.Get(commentsServiceURL + "/api/comments?news_id=" + newsID)
	if err != nil {
		log.Printf("Ошибка получения комментариев: %v", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Ошибка получения комментариев", http.StatusInternalServerError)
		return
	}
	defer commentsResp.Body.Close()

	if commentsResp.StatusCode != http.StatusOK {
		log.Printf("Ошибка получения комментариев: статус %d", commentsResp.StatusCode)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Ошибка получения комментариев", http.StatusInternalServerError)
		return
	}

	// Разбираем ответ с новостью
	var news NewsFullDetailed
	if err := json.NewDecoder(newsResp.Body).Decode(&news); err != nil {
		log.Printf("Ошибка разбора новости: %v", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Ошибка разбора новости", http.StatusInternalServerError)
		return
	}

	// Разбираем ответ с комментариями
	var comments []Comment
	if err := json.NewDecoder(commentsResp.Body).Decode(&comments); err != nil {
		log.Printf("Ошибка разбора комментариев: %v", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Ошибка разбора комментариев", http.StatusInternalServerError)
		return
	}

	// Объединяем новость и комментарии
	news.Comments = comments

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(news)
}

// handleAddComment обрабатывает добавление новых комментариев
func handleAddComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Ошибка чтения тела запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Проверяем комментарий через сервис цензуры
	censorResp, err := http.Post(censorshipServiceURL+"/api/censor", "application/json", bytes.NewReader(body))
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Ошибка проверки комментария", http.StatusInternalServerError)
		return
	}
	defer censorResp.Body.Close()

	if censorResp.StatusCode != http.StatusOK {
		// Копируем ответ об ошибке от сервиса цензуры
		for k, v := range censorResp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(censorResp.StatusCode)
		io.Copy(w, censorResp.Body)
		return
	}

	// Если комментарий прошел цензуру, создаем его
	resp, err := http.Post(commentsServiceURL+"/api/comments", "application/json", bytes.NewReader(body))
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Ошибка создания комментария", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Копируем заголовки ответа
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	// Копируем тело ответа
	io.Copy(w, resp.Body)
}
