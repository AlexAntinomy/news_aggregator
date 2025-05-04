package fetcher

import (
	"context"
	"fmt"
	"news_aggregator/internal/db"
	"news_aggregator/internal/logger"
	"news_aggregator/internal/models"
	"time"
)

// StartPolling запускает бесконечный цикл опроса RSS-лент из urls каждые interval.
// Каждая лента обрабатывается в отдельной горутине; результаты и ошибки передаются на агрегацию через каналы.
func StartPolling(ctx context.Context, database *db.Database, urls []string, interval time.Duration) {
	itemsCh := make(chan models.Item)
	errsCh := make(chan error)

	// Агрегатор: сохраняет новости и логирует ошибки
	go func() {
		for {
			select {
			case item := <-itemsCh:
				// Парсим дату из RSS и сохраняем в БД
				pubTime, err := time.Parse(time.RFC1123Z, item.PubDate)
				if err != nil {
					logger.Log.Warnf("Failed to parse date '%s': %v", item.PubDate, err)
					continue
				}
				err = database.SaveNewsItem(
					ctx,
					item.Title,
					item.Description,
					pubTime.Format(time.RFC3339),
					item.Link,
					item.FeedID,
				)
				if err != nil {
					logger.Log.Warnf("Failed to save news item: %v", err)
				}
			case err := <-errsCh:
				// Логируем ошибку обхода
				logger.Log.Errorf("Fetcher error: %v", err)
			case <-ctx.Done():
				// При завершении контекста завершаем агрегатор
				return
			}
		}
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Log.WithField("interval", interval.String()).Info("Starting new polling cycle")
			for _, url := range urls {
				go processFeed(ctx, database, url, itemsCh, errsCh)
			}
		case <-ctx.Done():
			logger.Log.Info("Stopping poller by context")
			return
		}
	}
}

// processFeed загружает одну RSS-ленту и отправляет каждый элемент в itemsCh, ошибки — в errsCh.
func processFeed(
	ctx context.Context,
	database *db.Database,
	url string,
	itemsCh chan<- models.Item,
	errsCh chan<- error,
) {
	logEntry := logger.Log.WithField("url", url)

	logEntry.Debug("Fetching RSS feed")
	rss, err := FetchRSS(url)
	if err != nil {
		errsCh <- fmt.Errorf("failed to fetch RSS %s: %w", url, err)
		return
	}

	logEntry = logEntry.WithField("items_count", len(rss.Channel.Items))
	logEntry.Info("Processing RSS feed")

	feedID, err := database.SaveFeed(ctx, url)
	if err != nil {
		errsCh <- fmt.Errorf("failed to save feed %s: %w", url, err)
		return
	}

	for _, itm := range rss.Channel.Items {
		// Наполняем FeedID и шлём в агрегатор
		itm.FeedID = feedID
		itemsCh <- itm
	}
}
