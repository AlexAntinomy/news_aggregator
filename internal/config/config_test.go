package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"news_aggregator/internal/config"

	"github.com/stretchr/testify/require"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
	return path
}

func TestLoadConfig_Success(t *testing.T) {
	json := `{
		"rss_feeds": ["https://example.com/rss", "http://foo.bar/feed"],
		"poll_interval": 10
	}`
	path := writeTempConfig(t, json)

	cfg, err := config.LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, []string{"https://example.com/rss", "http://foo.bar/feed"}, cfg.RSSFeeds)
	require.Equal(t, 10, cfg.PollInterval)
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := config.LoadConfig("/nonexistent/config.json")
	require.Error(t, err)
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	path := writeTempConfig(t, `{ invalid json }`)
	_, err := config.LoadConfig(path)
	require.Error(t, err)
}

func TestValidate_Success(t *testing.T) {
	cfg := &config.Config{
		RSSFeeds:     []string{"https://example.com/rss", "http://foo.bar/feed"},
		PollInterval: 5,
	}
	err := cfg.Validate()
	require.NoError(t, err)
}

func TestValidate_InvalidInterval(t *testing.T) {
	cfg := &config.Config{
		RSSFeeds:     []string{"https://example.com/rss"},
		PollInterval: 1,
	}
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "poll interval must be ≥ 5")
}

func TestValidate_InvalidURL(t *testing.T) {
	cfg := &config.Config{
		RSSFeeds:     []string{"not-a-url", "http://foo.bar/feed"},
		PollInterval: 5,
	}
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid RSS URL")
}
