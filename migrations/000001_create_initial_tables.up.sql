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

CREATE INDEX idx_news_publication_date ON news (publication_date DESC);