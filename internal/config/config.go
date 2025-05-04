package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
)

// Config хранит настройку списка RSS-лент и интервала опроса.
type Config struct {
	RSSFeeds     []string `json:"rss_feeds"`
	PollInterval int      `json:"poll_interval"`
}

// Validate проверяет, что PollInterval не меньше 5 секунд и все RSSFeeds — валидные URL.
func (cfg *Config) Validate() error {
	if cfg.PollInterval < 5 {
		return errors.New("poll interval must be ≥ 5 seconds")
	}
	for _, u := range cfg.RSSFeeds {
		if _, err := url.ParseRequestURI(u); err != nil {
			return fmt.Errorf("invalid RSS URL: %s", u)
		}
	}
	return nil
}

// LoadConfig читает JSON-файл по пути path, декодирует его в Config.
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
