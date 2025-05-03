package config

import (
	"encoding/json"
	"net/url"
	"os"
)

type RabbitMQConfig struct {
	URL   string `json:"url"`
	Queue string `json:"queue"`
}

type Config struct {
	RSSFeeds     []string       `json:"rss_feeds"`
	PollInterval int            `json:"poll_interval"`
	RabbitMQ     RabbitMQConfig `json:"rabbitmq"`
}

func (cfg *Config) Validate() error {
	for _, u := range cfg.RSSFeeds {
		if _, err := url.ParseRequestURI(u); err != nil {
			return err
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
