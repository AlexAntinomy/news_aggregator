package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
}

func NewDB(ctx context.Context, connString string) (*Database, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}
	return &Database{Pool: pool}, nil
}

func (db *Database) Close() {
	db.Pool.Close()
}

func (db *Database) SaveFeed(ctx context.Context, url string) (int, error) {
	var id int
	err := db.Pool.QueryRow(ctx, `
        INSERT INTO rss_feeds (url) 
        VALUES ($1)
        ON CONFLICT (url) DO UPDATE SET url = EXCLUDED.url
        RETURNING id
    `, url).Scan(&id)
	return id, err
}

func (db *Database) SaveNewsItem(ctx context.Context, title, description, pubDate, link string, feedID int) error {
	_, err := db.Pool.Exec(ctx, `
        INSERT INTO news (title, description, publication_date, source_link, rss_feed_id)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (source_link) DO NOTHING
    `, title, description, pubDate, link, feedID)
	return err
}
