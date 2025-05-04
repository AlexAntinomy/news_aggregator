-- Создание таблицы rss_feeds
CREATE TABLE IF NOT EXISTS rss_feeds (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы news
CREATE TABLE IF NOT EXISTS news (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    publication_date TIMESTAMPTZ NOT NULL,
    source_link TEXT NOT NULL,
    rss_feed_id INTEGER REFERENCES rss_feeds(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Добавление индекса для ускорения запросов
CREATE INDEX IF NOT EXISTS idx_news_rss_feed_id ON news(rss_feed_id);
