package main

import (
	"context"
	"log"
	"net/http"
	"news_aggregator/internal/config"
	"news_aggregator/internal/db"
	"news_aggregator/internal/fetcher"
	"news_aggregator/internal/server"
	"time"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database, err := db.NewDB(ctx, "postgres://admin:admin@localhost:5432/newsdb")
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer database.Close()

	go fetcher.StartPolling(
		ctx,
		database,
		cfg.RSSFeeds,
		time.Duration(cfg.PollInterval)*time.Minute,
	)

	srv := server.NewServer(database)
	http.HandleFunc("GET /api/news/{limit}", srv.GetNews)
	http.Handle("/", http.FileServer(http.Dir("./web")))

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
