-- Создание таблицы sources
CREATE TABLE IF NOT EXISTS sources (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы rss_feeds
CREATE TABLE IF NOT EXISTS rss_feeds (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    source_id INTEGER REFERENCES sources(id) ON DELETE CASCADE,
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

-- Добавление индексов для ускорения запросов
CREATE INDEX IF NOT EXISTS idx_news_rss_feed_id ON news(rss_feed_id);
CREATE INDEX IF NOT EXISTS idx_rss_feeds_source_id ON rss_feeds(source_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_news_source_link ON news(source_link);

-- Добавление начальных данных для источников
INSERT INTO sources (name) VALUES
    ('ТАСС'),
    ('Коммерсантъ'),
    ('Lenta.ru'),
    ('РИА Новости'),
    ('5-tv.ru')
ON CONFLICT (name) DO NOTHING;
