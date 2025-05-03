package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"news_aggregator/internal/db"
	"news_aggregator/internal/server"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const testConnString = "postgres://user:pass@localhost:5432/testdb?sslmode=disable"

func setupTestDB(t *testing.T) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), testConnString)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(), `
		DROP TABLE IF EXISTS news, rss_feeds CASCADE;
		CREATE TABLE rss_feeds (
			id SERIAL PRIMARY KEY,
			url VARCHAR(2048) UNIQUE NOT NULL,
			last_polled TIMESTAMP WITH TIME ZONE
		);
		
		CREATE TABLE news (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			publication_date TIMESTAMP WITH TIME ZONE NOT NULL,
			source_link VARCHAR(2048) UNIQUE NOT NULL,
			rss_feed_id INTEGER NOT NULL REFERENCES rss_feeds(id) ON DELETE CASCADE
		);
		
		INSERT INTO rss_feeds (url) VALUES ('https://testfeed.com/rss');
		INSERT INTO news (title, publication_date, source_link, rss_feed_id)
		VALUES ('Test Title', NOW(), 'http://test.com/1', 1);
	`)
	require.NoError(t, err)

	return pool
}

func TestGetNews(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	database := &db.Database{Pool: pool}
	srv := server.NewServer(database)

	t.Run("valid request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/news/1", nil)
		w := httptest.NewRecorder()

		srv.GetNews(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), "Test Title")
	})
}
