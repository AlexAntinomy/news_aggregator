package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Database инкапсулирует пул соединений к PostgreSQL.
type Database struct {
	Pool *pgxpool.Pool
}

// NewDB создаёт новый пул соединений по connString и возвращает Database.
func NewDB(ctx context.Context, connString string) (*Database, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}
	return &Database{Pool: pool}, nil
}

// Close закрывает пул соединений.
func (db *Database) Close() {
	db.Pool.Close()
}

// SaveFeed сохраняет URL RSS-ленты в таблицу rss_feeds и возвращает её id.
// В случае конфликта по URL выполняется обновление (чтобы вернуть существующий id).
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

// SaveNewsItem сохраняет один элемент новости в таблицу news.
// Если запись с таким source_link уже есть, то операция игнорируется.
func (db *Database) SaveNewsItem(ctx context.Context, title, description, pubDate, link string, feedID int) error {
	_, err := db.Pool.Exec(ctx, `
        INSERT INTO news (title, description, publication_date, source_link, rss_feed_id)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (source_link) DO NOTHING
    `, title, description, pubDate, link, feedID)
	return err
}
