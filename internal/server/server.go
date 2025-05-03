package server

import (
	"encoding/json"
	"net/http"
	"news_aggregator/internal/db"
	"strconv"
	"time"
)

type Server struct {
	db *db.Database
}

func NewServer(db *db.Database) *Server {
	return &Server{db: db}
}

func (s *Server) GetNews(w http.ResponseWriter, r *http.Request) {
	limitStr := r.PathValue("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
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
			title, desc, link, feedURL string
			pubDate                    time.Time
		)

		if err := rows.Scan(&title, &desc, &pubDate, &link, &feedURL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		news = append(news, map[string]interface{}{
			"title":       title,
			"description": desc,
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
