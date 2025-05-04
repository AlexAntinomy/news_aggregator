package main

import (
	"context"
	"net/http"
	"news_aggregator/internal/config"
	"news_aggregator/internal/db"
	"news_aggregator/internal/fetcher"
	"news_aggregator/internal/logger"
	"news_aggregator/internal/server"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger.Init()
	defer logger.Log.Info("Application stopped")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Log.Info("Loading configuration")
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		logger.Log.Fatalf("Failed to load config: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		logger.Log.Fatalf("Invalid configuration: %v", err)
	}

	// Получаем строку подключения из переменной окружения
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Log.Fatal("DATABASE_URL is not set")
	}

	logger.Log.Info("Connecting to database")
	database, err := db.NewDB(ctx, dbURL)
	if err != nil {
		logger.Log.Fatalf("Database connection error: %v", err)
	}
	defer func() {
		database.Close()
		logger.Log.Info("Database connection closed")
	}()

	logger.Log.WithField("interval", "5s").Info("Starting RSS polling")
	go fetcher.StartPolling(
		ctx,
		database,
		cfg.RSSFeeds,
		5*time.Second,
	)

	srv := server.NewServer(database)

	http.HandleFunc("/api/news/count", srv.GetNewNewsCount)
	http.HandleFunc("/api/news/", srv.GetNews)
	http.HandleFunc("/health", srv.HealthCheck)
	http.Handle("/", http.FileServer(http.Dir("./web")))

	server := &http.Server{Addr: ":8080"}
	go func() {
		logger.Log.Info("Starting HTTP server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatalf("Server fatal error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Received shutdown signal")
	ctxShutdown, cancelShutdown := context.WithTimeout(ctx, 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Log.Fatalf("Forced shutdown: %v", err)
	}
	logger.Log.Info("Server stopped gracefully")
}
