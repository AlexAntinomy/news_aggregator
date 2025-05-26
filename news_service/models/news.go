package models

import "time"

// News represents a news item in our system
type News struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	PublishedAt time.Time `json:"published_at"`
	SourceURL   string    `json:"source_url"`
	SourceName  string    `json:"source_name"`
}
