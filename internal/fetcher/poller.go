package fetcher

import (
	"context"
	"news_aggregator/internal/db"
	"news_aggregator/internal/logger"
	"news_aggregator/internal/models"
	"time"
)

func StartPolling(ctx context.Context, db *db.Database, urls []string, interval time.Duration) {
	log := logger.Log.WithFields(map[string]interface{}{
		"service":  "poller",
		"interval": interval.String(),
	})

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Info("Starting new polling cycle")
			for _, url := range urls {
				go processFeed(ctx, db, url)
			}

		case <-ctx.Done():
			log.Info("Stopping poller by context")
			return
		}
	}
}

func processFeed(ctx context.Context, db *db.Database, url string) {
	log := logger.Log.WithField("url", url)

	log.Debug("Fetching RSS feed")
	rss, err := FetchRSS(url)
	if err != nil {
		log.Errorf("Failed to fetch RSS: %v", err)
		return
	}

	log = log.WithField("items_count", len(rss.Channel.Items))
	log.Info("Processing RSS feed")

	feedID, err := db.SaveFeed(ctx, url)
	if err != nil {
		log.Errorf("Failed to save feed: %v", err)
		return
	}

	for _, item := range rss.Channel.Items {
		processItem(ctx, db, log, item, feedID)
	}
}

func processItem(ctx context.Context, db *db.Database, log *logger.Entry, item models.Item, feedID int) {
	pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
	if err != nil {
		log.Warnf("Failed to parse date '%s': %v", item.PubDate, err)
		return
	}

	if err := db.SaveNewsItem(
		ctx,
		item.Title,
		item.Description,
		pubDate.Format(time.RFC3339),
		item.Link,
		feedID,
	); err != nil {
		log.Warnf("Failed to save news item: %v", err)
	}
}
