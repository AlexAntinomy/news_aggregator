package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"news_aggregator/news_service/middleware"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// RSS представляет структуру RSS-ленты
type RSS struct {
	Channel Channel `xml:"channel"`
}

// Channel представляет элемент канала в RSS
type Channel struct {
	Title string `xml:"title"`
	Items []Item `xml:"item"`
}

// Item представляет новость в RSS
type Item struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Link        string `xml:"link"`
}

// News представляет новость в нашей системе
type News struct {
	ID              int       `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	PublicationDate time.Time `json:"date"`
	SourceLink      string    `json:"source_link"`
	SourceName      string    `json:"source"`
}

// Добавляю структуру для конфига
type Config struct {
	RSSFeeds     []string `json:"rss_feeds"`
	PollInterval int      `json:"poll_interval"`
}

const (
	defaultPageSize = 15
	maxPageSize     = 100
	// Настройки HTTP-клиента
	httpTimeout = 10 * time.Second
	maxRetries  = 3
	retryDelay  = 2 * time.Second
)

var (
	// Метрики Prometheus
	newsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "news_total",
		Help: "Total number of news items processed",
	})

	feedFetchDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "feed_fetch_duration_seconds",
		Help:    "Time spent fetching RSS feeds",
		Buckets: prometheus.DefBuckets,
	})

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// Глобальные переменные
	db     *pgxpool.Pool
	logger *logrus.Logger
)

func init() {
	// Регистрируем метрики
	prometheus.MustRegister(newsTotal)
	prometheus.MustRegister(feedFetchDuration)
	prometheus.MustRegister(httpRequestDuration)
}

func main() {
	// Настраиваем логгер
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	// Получаем уровень логирования из переменной окружения
	if level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL")); err == nil {
		logger.SetLevel(level)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Создаем канал для сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Подключаемся к базе данных
	var err error
	db, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()

	// Проверяем соединение с базой данных
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		logger.Fatalf("Unable to ping database: %v", err)
	}
	logger.Info("Successfully connected to database")

	// Читаем config.json
	var config Config
	configFile, err := os.Open("config.json")
	if err != nil {
		logger.Fatalf("Не удалось открыть config.json: %v", err)
	}
	defer configFile.Close()
	if err := json.NewDecoder(configFile).Decode(&config); err != nil {
		logger.Fatalf("Не удалось прочитать config.json: %v", err)
	}
	if len(config.RSSFeeds) == 0 {
		logger.Warn("Список RSS-лент пуст в config.json")
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 5 // по умолчанию 5 минут
	}

	// Создаем HTTP сервер
	mux := http.NewServeMux()

	// Добавляем обработчики
	mux.HandleFunc("/api/news", handleNewsList)
	mux.HandleFunc("/api/news/", handleNewsDetail)
	mux.HandleFunc("/health", handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	// Применяем middleware
	handler := middleware.RequestIDMiddleware(mux)
	handler = middleware.LoggingMiddleware(handler, logger)
	handler = middleware.MetricsMiddleware(handler, httpRequestDuration)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Запускаем сервер в горутине
	go func() {
		logger.Info("Starting server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server error: %v", err)
		}
	}()

	// Первая загрузка новостей сразу при старте
	if err := fetchAndSaveFeed(db, logger, config.RSSFeeds); err != nil {
		logger.Errorf("Error fetching feed: %v", err)
	}

	// Запускаем периодическое обновление новостей
	go func() {
		ticker := time.NewTicker(time.Duration(config.PollInterval) * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := fetchAndSaveFeed(db, logger, config.RSSFeeds); err != nil {
					logger.Errorf("Error fetching feed: %v", err)
				}
			}
		}
	}()

	// Ждем сигнала завершения
	<-sigChan
	logger.Info("Shutting down server...")

	// Создаем контекст с таймаутом для graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Закрываем сервер
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server stopped")
}

// handleHealth обрабатывает запросы к /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем соединение с базой данных
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		http.Error(w, "Database connection failed", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}
}

func fetchAndSaveFeed(db *pgxpool.Pool, logger *logrus.Logger, feeds []string) error {
	start := time.Now()
	defer func() {
		feedFetchDuration.Observe(time.Since(start).Seconds())
	}()

	for _, feedURL := range feeds {
		go func(url string) {
			var lastErr error
			for attempt := 1; attempt <= maxRetries; attempt++ {
				// Создаем контекст с таймаутом
				ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
				defer cancel()

				// Создаем запрос с контекстом
				req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
				if err != nil {
					logger.WithError(err).WithField("url", url).Error("Ошибка создания запроса")
					return
				}

				// Выполняем запрос
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					lastErr = err
					logger.WithError(err).WithFields(logrus.Fields{
						"url":     url,
						"attempt": attempt,
					}).Warn("Ошибка получения RSS-ленты")
					if attempt < maxRetries {
						time.Sleep(retryDelay)
						return
					}
					return
				}
				defer resp.Body.Close()

				// Проверяем статус ответа
				if resp.StatusCode != http.StatusOK {
					lastErr = fmt.Errorf("неверный статус ответа: %d", resp.StatusCode)
					logger.WithError(lastErr).WithFields(logrus.Fields{
						"url":     url,
						"status":  resp.StatusCode,
						"attempt": attempt,
					}).Warn("Ошибка получения RSS-ленты")
					if attempt < maxRetries {
						time.Sleep(retryDelay)
						return
					}
					return
				}

				// Проверяем Content-Type
				contentType := resp.Header.Get("Content-Type")
				if !strings.Contains(contentType, "xml") && !strings.Contains(contentType, "rss") {
					lastErr = fmt.Errorf("неверный Content-Type: %s", contentType)
					logger.WithError(lastErr).WithFields(logrus.Fields{
						"url":         url,
						"contentType": contentType,
						"attempt":     attempt,
					}).Warn("Неверный тип контента")
					if attempt < maxRetries {
						time.Sleep(retryDelay)
						return
					}
					return
				}

				// Читаем и декодируем ответ
				var rss RSS
				if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
					lastErr = err
					logger.WithError(err).WithFields(logrus.Fields{
						"url":     url,
						"attempt": attempt,
					}).Warn("Ошибка декодирования RSS")
					if attempt < maxRetries {
						time.Sleep(retryDelay)
						return
					}
					return
				}

				// Если все успешно, сохраняем данные
				if err := saveFeedData(db, url, rss); err != nil {
					logger.WithError(err).WithField("url", url).Error("Ошибка сохранения данных")
					newsTotal.Inc()
				} else {
					newsTotal.Inc()
				}
				return
			}

			// Если все попытки не удались
			logger.WithError(lastErr).WithFields(logrus.Fields{
				"url":      url,
				"attempts": maxRetries,
			}).Error("Не удалось получить RSS-ленту после всех попыток")
			newsTotal.Inc()
		}(feedURL)
	}

	return nil
}

// saveFeedData сохраняет данные из RSS-ленты в базу данных
func saveFeedData(db *pgxpool.Pool, feedURL string, rss RSS) error {
	// Определяем источник на основе URL
	var sourceName string
	switch {
	case strings.Contains(feedURL, "tass.ru"):
		sourceName = "ТАСС"
	case strings.Contains(feedURL, "kommersant.ru"):
		sourceName = "Коммерсантъ"
	case strings.Contains(feedURL, "lenta.ru"):
		sourceName = "Lenta.ru"
	case strings.Contains(feedURL, "ria.ru"):
		sourceName = "РИА Новости"
	case strings.Contains(feedURL, "5-tv.ru"):
		sourceName = "5-tv.ru"
	default:
		sourceName = "Неизвестный источник"
	}

	// Создаем транзакцию
	tx, err := db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("ошибка создания транзакции: %v", err)
	}
	defer tx.Rollback(context.Background())

	// Получаем или создаем источник
	var sourceID int
	err = tx.QueryRow(context.Background(), `
		INSERT INTO sources (name)
		VALUES ($1)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, sourceName).Scan(&sourceID)
	if err != nil {
		return fmt.Errorf("ошибка сохранения источника %s: %v", sourceName, err)
	}

	// Сохраняем или обновляем ленту
	var feedID int
	err = tx.QueryRow(context.Background(), `
		INSERT INTO rss_feeds (url, source_id)
		VALUES ($1, $2)
		ON CONFLICT (url) DO UPDATE SET source_id = EXCLUDED.source_id
		RETURNING id
	`, feedURL, sourceID).Scan(&feedID)
	if err != nil {
		return fmt.Errorf("ошибка сохранения ленты %s: %v", feedURL, err)
	}

	// Сохраняем новости
	for _, item := range rss.Channel.Items {
		pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"url":   feedURL,
				"date":  item.PubDate,
				"title": item.Title,
			}).Warn("Ошибка разбора даты")
			continue
		}

		_, err = tx.Exec(context.Background(), `
			INSERT INTO news (title, description, publication_date, source_link, rss_feed_id)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (source_link) DO NOTHING
		`, item.Title, item.Description, pubDate, item.Link, feedID)
		if err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"url":   feedURL,
				"title": item.Title,
			}).Warn("Ошибка сохранения новости")
		}
	}

	// Завершаем транзакцию
	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("ошибка завершения транзакции: %v", err)
	}

	return nil
}

func handleNewsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Разбираем параметры пагинации
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("page_size"))
	if err != nil || pageSize < 1 || pageSize > maxPageSize {
		pageSize = defaultPageSize
	}

	search := r.URL.Query().Get("s")

	// Вычисляем смещение
	offset := (page - 1) * pageSize

	// Формируем базовый запрос
	baseQuery := `
		SELECT n.id, n.title, n.description, n.publication_date, n.source_link, s.name as source_name
		FROM news n
		JOIN rss_feeds rf ON n.rss_feed_id = rf.id
		JOIN sources s ON rf.source_id = s.id
	`

	// Формируем базовый запрос для подсчета общего количества
	baseCountQuery := `
		SELECT COUNT(*)
		FROM news n
		JOIN rss_feeds rf ON n.rss_feed_id = rf.id
		JOIN sources s ON rf.source_id = s.id
	`

	var query, countQuery string
	var args []interface{}
	argCount := 1

	if search != "" {
		query = baseQuery + " WHERE n.title ILIKE $" + strconv.Itoa(argCount)
		countQuery = baseCountQuery + " WHERE n.title ILIKE $" + strconv.Itoa(argCount)
		args = append(args, "%"+search+"%")
		argCount++
	} else {
		query = baseQuery
		countQuery = baseCountQuery
	}

	query += " ORDER BY n.publication_date DESC LIMIT $" + strconv.Itoa(argCount) + " OFFSET $" + strconv.Itoa(argCount+1)
	args = append(args, pageSize, offset)

	// Получаем общее количество записей
	var totalItems int
	err = db.QueryRow(r.Context(), countQuery, args[:argCount-1]...).Scan(&totalItems)
	if err != nil {
		logger.WithError(err).Error("Ошибка получения общего количества записей")
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Получаем записи с пагинацией
	rows, err := db.Query(r.Context(), query, args...)
	if err != nil {
		http.Error(w, "Ошибка получения новостей", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var news []News
	for rows.Next() {
		var n News
		if err := rows.Scan(&n.ID, &n.Title, &n.Description, &n.PublicationDate, &n.SourceLink, &n.SourceName); err != nil {
			http.Error(w, "Ошибка сканирования новостей", http.StatusInternalServerError)
			return
		}
		news = append(news, n)
	}

	response := struct {
		Items      []News `json:"items"`
		Pagination struct {
			TotalItems   int `json:"total_items"`
			TotalPages   int `json:"total_pages"`
			CurrentPage  int `json:"current_page"`
			ItemsPerPage int `json:"items_per_page"`
		} `json:"pagination"`
	}{
		Items: news,
	}
	response.Pagination.TotalItems = totalItems
	response.Pagination.TotalPages = (totalItems + pageSize - 1) / pageSize
	response.Pagination.CurrentPage = page
	response.Pagination.ItemsPerPage = pageSize

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleNewsDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 {
		http.Error(w, "Неверный ID новости", http.StatusBadRequest)
		return
	}

	newsID := parts[3]
	var news News
	err := db.QueryRow(context.Background(), `
		SELECT n.id, n.title, n.description, n.publication_date, n.source_link, s.name as source_name
		FROM news n
		JOIN rss_feeds rf ON n.rss_feed_id = rf.id
		JOIN sources s ON rf.source_id = s.id
		WHERE n.id = $1
	`, newsID).Scan(&news.ID, &news.Title, &news.Description, &news.PublicationDate, &news.SourceLink, &news.SourceName)
	if err != nil {
		logger.WithError(err).Error("Ошибка получения деталей новости")
		http.Error(w, "Новость не найдена", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(news)
}

func getSourceFromURL(url string) string {
	switch {
	case strings.Contains(url, "tass.ru"):
		return "ТАСС"
	case strings.Contains(url, "kommersant.ru"):
		return "Коммерсантъ"
	case strings.Contains(url, "lenta.ru"):
		return "Lenta.ru"
	case strings.Contains(url, "ria.ru"):
		return "РИА Новости"
	case strings.Contains(url, "5-tv.ru"):
		return "5-tv.ru"
	default:
		return "Неизвестный источник"
	}
}
