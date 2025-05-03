package fetcher_test

import (
	"net/http"
	"net/http/httptest"
	"news_aggregator/internal/fetcher"
	"news_aggregator/internal/models"
	"testing"
)

func TestFetchRSS(t *testing.T) {
	testCases := []struct {
		name     string
		xml      string
		expected models.RSS
		wantErr  bool
	}{
		{
			name: "valid rss",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<rss version="2.0">
				<channel>
					<title>Test Feed</title>
					<item>
						<title>Test Title</title>
						<description>Test Description</description>
						<pubDate>Wed, 03 May 2023 15:04:05 +0000</pubDate>
						<link>http://example.com/test</link>
					</item>
				</channel>
			</rss>`,
			expected: models.RSS{
				Channel: models.Channel{
					Title: "Test Feed",
					Items: []models.Item{
						{
							Title:       "Test Title",
							Description: "Test Description",
							PubDate:     "Wed, 03 May 2023 15:04:05 +0000",
							Link:        "http://example.com/test",
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(tc.xml))
			}))
			defer server.Close()

			result, err := fetcher.FetchRSS(server.URL)
			if (err != nil) != tc.wantErr {
				t.Fatalf("FetchRSS() error = %v, wantErr %v", err, tc.wantErr)
			}

			if result.Channel.Title != tc.expected.Channel.Title {
				t.Errorf("Expected title %q, got %q", tc.expected.Channel.Title, result.Channel.Title)
			}

			if len(result.Channel.Items) != len(tc.expected.Channel.Items) {
				t.Fatalf("Expected %d items, got %d", len(tc.expected.Channel.Items), len(result.Channel.Items))
			}

			for i, item := range result.Channel.Items {
				if item.Title != tc.expected.Channel.Items[i].Title {
					t.Errorf("Item %d title mismatch: expected %q, got %q", i, tc.expected.Channel.Items[i].Title, item.Title)
				}
			}
		})
	}
}
