package worker

import (
	"context"
	"news_aggregator/internal/db"
	"news_aggregator/internal/fetcher"
	"news_aggregator/internal/logger"
	"time"
)

type Worker struct {
	db *db.Database
}

func NewWorker(db *db.Database) *Worker {
	return &Worker{db: db}
}

func (w *Worker) HandleTask(body []byte) error {
	ctx := context.Background()
	url := string(body)

	log := logger.Log.WithField("url", url)
	log.Info("Processing RSS feed")

	rss, err := fetcher.FetchRSS(url)
	if err != nil {
		log.Errorf("Fetch failed: %v", err)
		return err
	}

	feedID, err := w.db.SaveFeed(ctx, url)
	if err != nil {
		log.Errorf("Save feed failed: %v", err)
		return err
	}

	for _, item := range rss.Channel.Items {
		pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			log.Warnf("Parse date failed: %v", err)
			continue
		}

		if err := w.db.SaveNewsItem(
			ctx,
			item.Title,
			item.Description,
			pubDate.Format(time.RFC3339),
			item.Link,
			feedID,
		); err != nil {
			log.Warnf("Save item failed: %v", err)
		}
	}

	log.Infof("Processed %d items", len(rss.Channel.Items))
	return nil
}
