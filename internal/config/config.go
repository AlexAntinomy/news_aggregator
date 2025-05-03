package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	RSSFeeds     []string `json:"rss_feeds"`
	PollInterval int      `json:"poll_interval"`
}

func (cfg *Config) Validate() error {
	if cfg.PollInterval < 1 {
		return errors.New("poll interval must be ≥ 1 minute")
	}
	for _, u := range cfg.RSSFeeds {
		if _, err := url.ParseRequestURI(u); err != nil {
			return fmt.Errorf("invalid RSS URL: %s", u)
		}
	}
	return nil
}

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
