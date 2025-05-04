package fetcher

import (
	"encoding/xml"
	"net/http"
	"news_aggregator/internal/models"
	"time"
)

// FetchRSS загружает XML-ленты по url, декодирует и возвращает структуру models.RSS.
func FetchRSS(url string) (*models.RSS, error) {
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rss models.RSS
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, err
	}
	return &rss, nil
}
