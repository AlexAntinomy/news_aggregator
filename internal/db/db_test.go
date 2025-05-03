package db_test

import (
	"context"
	"news_aggregator/internal/db"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const testConnString = "postgres://user:pass@localhost:5432/testdb?sslmode=disable"

func setupTestDB(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, testConnString)
	require.NoError(t, err)

	// Применяем миграции
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS rss_feeds (
			id SERIAL PRIMARY KEY,
			url VARCHAR(2048) UNIQUE NOT NULL,
			last_polled TIMESTAMP WITH TIME ZONE
		);
		
		CREATE TABLE IF NOT EXISTS news (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			publication_date TIMESTAMP WITH TIME ZONE NOT NULL,
			source_link VARCHAR(2048) UNIQUE NOT NULL,
			rss_feed_id INTEGER NOT NULL REFERENCES rss_feeds(id) ON DELETE CASCADE
		);
		
		TRUNCATE TABLE news, rss_feeds RESTART IDENTITY;
	`)
	require.NoError(t, err)

	return pool
}

func TestSaveFeed(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	database := &db.Database{Pool: pool}

	t.Run("save new feed", func(t *testing.T) {
		id, err := database.SaveFeed(context.Background(), "https://test.com/rss")
		require.NoError(t, err)
		require.Equal(t, 1, id)
	})
}
