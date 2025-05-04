package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"news_aggregator/internal/db"
	"strconv"
	"time"
)

// Server хранит зависимости HTTP-обработчиков, в частности БД.
type Server struct {
	db *db.Database
}

// NewServer создаёт новый экземпляр Server с переданной базой данных.
func NewServer(db *db.Database) *Server {
	return &Server{db: db}
}

// HealthCheck отвечает 200 OK, если база доступна, иначе 503.
func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Pool.Ping(r.Context()); err != nil {
		http.Error(w, "DB unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("OK"))
}

// GetNews возвращает JSON-массив последних limit новостей, сортированных по дате.
func (s *Server) GetNews(w http.ResponseWriter, r *http.Request) {
	// извлекаем limit из пути /api/news/{limit}
	path := r.URL.Path[len("/api/news/"):]
	limit, err := strconv.Atoi(path)
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := s.db.Pool.Query(r.Context(), `
        SELECT n.title, n.description, n.publication_date, n.source_link, r.url 
        FROM news n
        JOIN rss_feeds r ON n.rss_feed_id = r.id
        ORDER BY publication_date DESC
        LIMIT $1
    `, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var news []map[string]interface{}
	for rows.Next() {
		var (
			title, link, feedURL string
			pubDate              time.Time
			desc                 sql.NullString
		)

		if err := rows.Scan(&title, &desc, &pubDate, &link, &feedURL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		news = append(news, map[string]interface{}{
			"title":       title,
			"description": desc.String,
			"date":        pubDate.Format(time.RFC3339),
			"link":        link,
			"feed":        feedURL,
		})
	}

	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(news); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// GetNewNewsCount возвращает JSON {"count": N} с количеством новостей,
// опубликованных после времени since в параметре запроса.
func (s *Server) GetNewNewsCount(w http.ResponseWriter, r *http.Request) {
	lastUpdateStr := r.URL.Query().Get("since")
	lastUpdate, err := time.Parse(time.RFC3339, lastUpdateStr)
	if err != nil {
		http.Error(w, "Invalid time format", http.StatusBadRequest)
		return
	}

	var count int
	err = s.db.Pool.QueryRow(r.Context(), `
        SELECT COUNT(*) 
        FROM news 
        WHERE publication_date > $1
    `, lastUpdate).Scan(&count)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}
