package fetcher

import (
	"context"
	"log"
	"news_aggregator/internal/db"
	"time"
)

func StartPolling(ctx context.Context, db *db.Database, urls []string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, url := range urls {
				go func(url string) {
					rss, err := FetchRSS(url)
					if err != nil {
						log.Printf("Error fetching %s: %v", url, err)
						return
					}

					feedID, err := db.SaveFeed(ctx, url)
					if err != nil {
						log.Printf("Error saving feed: %v", err)
						return
					}

					for _, item := range rss.Channel.Items {
						pubDate, _ := time.Parse(time.RFC1123Z, item.PubDate)
						if err := db.SaveNewsItem(ctx, item.Title, item.Description, pubDate.Format(time.RFC3339), item.Link, feedID); err != nil {
							log.Printf("Error saving news item: %v", err)
						}
					}
				}(url)
			}
		case <-ctx.Done():
			return
		}
	}
}
