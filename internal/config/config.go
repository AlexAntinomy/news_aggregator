package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	RSSFeeds     []string `json:"rss_feeds"`
	PollInterval int      `json:"poll_interval"`
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
