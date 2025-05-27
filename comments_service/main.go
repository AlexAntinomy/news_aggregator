package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// Комментарий к новости
type Comment struct {
	ID        int       `json:"id"`
	NewsID    int       `json:"news_id"`
	ParentID  *int      `json:"parent_id,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Тело запроса для создания комментария
type CommentRequest struct {
	NewsID   int    `json:"news_id"`
	ParentID *int   `json:"parent_id,omitempty"`
	Content  string `json:"content"`
}

var db *pgxpool.Pool

func main() {
	// Настройка логгера
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	// Инициализация подключения к базе данных
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUser == "" {
		dbUser = "postgres"
	}
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	if dbName == "" {
		dbName = "comments_db"
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	db, err = pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer db.Close()

	// Создание таблиц, если они не существуют
	if err := createTables(); err != nil {
		log.Fatalf("Ошибка создания таблиц: %v", err)
	}

	// Настройка HTTP-обработчиков с middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/comments", handleComments)

	// Применение middleware
	handler := RequestIDMiddleware(mux)
	handler = LoggingMiddleware(handler)

	log.Printf("Запуск сервиса комментариев на порту :8081")
	if err := http.ListenAndServe(":8081", handler); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func createTables() error {
	_, err := db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS comments (
			id SERIAL PRIMARY KEY,
			news_id INTEGER NOT NULL,
			parent_id INTEGER REFERENCES comments(id),
			content TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		w.Write([]byte(`{"status": "healthy"}`))
	}
}

func handleComments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetComments(w, r)
	case http.MethodPost:
		handleCreateComment(w, r)
	default:
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}
}

func handleGetComments(w http.ResponseWriter, r *http.Request) {
	newsID := r.URL.Query().Get("news_id")
	if newsID == "" {
		http.Error(w, "Требуется ID новости", http.StatusBadRequest)
		return
	}

	comments, err := getCommentsByNewsID(newsID)
	if err != nil {
		http.Error(w, "Ошибка получения комментариев", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func handleCreateComment(w http.ResponseWriter, r *http.Request) {
	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверное тело запроса", http.StatusBadRequest)
		return
	}

	comment, err := createComment(req)
	if err != nil {
		http.Error(w, "Ошибка создания комментария", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

func getCommentsByNewsID(newsID string) ([]Comment, error) {
	rows, err := db.Query(context.Background(), `
		SELECT id, news_id, parent_id, content, created_at
		FROM comments
		WHERE news_id = $1
		ORDER BY created_at DESC
	`, newsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.NewsID, &parentID, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			c.ParentID = &pid
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func createComment(req CommentRequest) (*Comment, error) {
	var comment Comment
	err := db.QueryRow(context.Background(), `
		INSERT INTO comments (news_id, parent_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, news_id, parent_id, content, created_at
	`, req.NewsID, req.ParentID, req.Content).Scan(
		&comment.ID, &comment.NewsID, &comment.ParentID, &comment.Content, &comment.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// Middleware для добавления ID запроса в контекст
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

// Middleware для логирования информации о каждом запросе
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
		}).Info("Request processed")
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
	rand.Read(b)
	return hex.EncodeToString(b)
}
