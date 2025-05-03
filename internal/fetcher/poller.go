package fetcher

import (
	"context"
	"news_aggregator/internal/logger"
	"news_aggregator/internal/queue"
	"time"
)

func StartPolling(
	ctx context.Context,
	producer *queue.Producer,
	urls []string,
	interval time.Duration,
	queueName string, // Добавлен параметр имени очереди
) {
	log := logger.Log.WithField("component", "poller")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Info("Starting new polling cycle")
			for _, url := range urls {
				// Используем queueName из конфига
				if err := producer.Publish(queueName, []byte(url)); err != nil {
					log.Errorf("Failed to publish URL '%s': %v", url, err)
					continue
				}
				log.Debugf("Published to '%s': %s", queueName, url)
			}

		case <-ctx.Done():
			log.Info("Polling stopped by context")
			return
		}
	}
}
