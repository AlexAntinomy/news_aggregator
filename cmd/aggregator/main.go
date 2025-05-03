package main

import (
	"context"
	"net/http"
	"news_aggregator/internal/config"
	"news_aggregator/internal/db"
	"news_aggregator/internal/fetcher"
	"news_aggregator/internal/logger"
	"news_aggregator/internal/queue"
	"news_aggregator/internal/server"
	"news_aggregator/internal/worker"
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

	// Загрузка конфигурации
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		logger.Log.Fatalf("Config load error: %v", err)
	}

	// Инициализация БД
	database, err := db.NewDB(ctx, "postgres://admin:admin@localhost:5432/newsdb")
	if err != nil {
		logger.Log.Fatalf("DB connection error: %v", err)
	}
	defer database.Close()

	// Настройка RabbitMQ Producer
	producer, err := queue.NewProducer(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Log.Fatalf("RabbitMQ producer error: %v", err)
	}
	defer producer.Close()

	// Настройка RabbitMQ Consumer
	consumer, err := queue.NewConsumer(
		cfg.RabbitMQ.URL,
		cfg.RabbitMQ.Queue,
		5, // Количество воркеров
	)
	if err != nil {
		logger.Log.Fatalf("RabbitMQ consumer error: %v", err)
	}
	defer consumer.Close()

	// Запуск воркеров
	wrk := worker.NewWorker(database)
	consumer.Consume(func(body []byte) error {
		return wrk.HandleTask(body)
	})

	// Запуск периодического опроса
	go fetcher.StartPolling(
		ctx,
		producer,
		cfg.RSSFeeds,
		time.Duration(cfg.PollInterval)*time.Minute,
		cfg.RabbitMQ.Queue, // Добавлен 5-й аргумент - имя очереди
	)

	// HTTP сервер
	srv := server.NewServer(database)
	http.HandleFunc("GET /api/news/{limit}", srv.GetNews)
	http.HandleFunc("GET /health", srv.HealthCheck)
	http.Handle("/", http.FileServer(http.Dir("./web")))

	server := &http.Server{Addr: ":8080"}
	go func() {
		logger.Log.Info("Starting HTTP server on :8080")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			logger.Log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down...")
	ctxShutdown, cancelShutdown := context.WithTimeout(ctx, 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Log.Fatalf("Forced shutdown: %v", err)
	}
}
