package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"news_service/middleware"

	"github.com/jackc/pgx/v5/pgxpool"
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
	Content         string    `json:"content"`
	Description     string    `json:"description"`
	PublicationDate time.Time `json:"date"`
	SourceLink      string    `json:"source_link"`
	SourceName      string    `json:"source"`
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
	db *pgxpool.Pool
	// Создаем HTTP-клиент с таймаутами
	httpClient = &http.Client{
		Timeout: httpTimeout,
	}
)

func main() {
	// Настройка логгера
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	// Инициализация подключения к базе данных
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUser == "" {
		dbUser = "postgres"
	}
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	if dbName == "" {
		dbName = "news_db"
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	db, err = pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer db.Close()

	// Запуск опроса RSS
	go startPolling()

	// Настройка HTTP-обработчиков с middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/api/news", handleNewsList)
	mux.HandleFunc("/api/news/", handleNewsDetail)

	// Применяем middleware
	handler := middleware.RequestIDMiddleware(mux)
	handler = middleware.LoggingMiddleware(handler)

	log.Println("Запуск сервиса новостей на порту :8082")
	if err := http.ListenAndServe(":8082", handler); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func startPolling() {
	// URL RSS-лент
	feeds := []string{
		"https://tass.ru/rss/v2.xml",
		"https://www.kommersant.ru/RSS/news.xml",
		"https://lenta.ru/rss",
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		for _, feedURL := range feeds {
			go fetchAndSaveFeed(feedURL)
		}
		<-ticker.C
	}
}

func fetchAndSaveFeed(feedURL string) {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Создаем контекст с таймаутом
		ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
		defer cancel()

		// Создаем запрос с контекстом
		req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
		if err != nil {
			logrus.WithError(err).WithField("url", feedURL).Error("Ошибка создания запроса")
			return
		}

		// Выполняем запрос
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			logrus.WithError(err).WithFields(logrus.Fields{
				"url":     feedURL,
				"attempt": attempt,
			}).Warn("Ошибка получения RSS-ленты")
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return
		}
		defer resp.Body.Close()

		// Проверяем статус ответа
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("неверный статус ответа: %d", resp.StatusCode)
			logrus.WithError(lastErr).WithFields(logrus.Fields{
				"url":     feedURL,
				"status":  resp.StatusCode,
				"attempt": attempt,
			}).Warn("Ошибка получения RSS-ленты")
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return
		}

		// Проверяем Content-Type
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "xml") && !strings.Contains(contentType, "rss") {
			lastErr = fmt.Errorf("неверный Content-Type: %s", contentType)
			logrus.WithError(lastErr).WithFields(logrus.Fields{
				"url":         feedURL,
				"contentType": contentType,
				"attempt":     attempt,
			}).Warn("Неверный тип контента")
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return
		}

		// Читаем и декодируем ответ
		var rss RSS
		if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
			lastErr = err
			logrus.WithError(err).WithFields(logrus.Fields{
				"url":     feedURL,
				"attempt": attempt,
			}).Warn("Ошибка декодирования RSS")
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return
		}

		// Если все успешно, сохраняем данные
		if err := saveFeedData(feedURL, rss); err != nil {
			logrus.WithError(err).WithField("url", feedURL).Error("Ошибка сохранения данных")
		}
		return
	}

	// Если все попытки не удались
	logrus.WithError(lastErr).WithFields(logrus.Fields{
		"url":      feedURL,
		"attempts": maxRetries,
	}).Error("Не удалось получить RSS-ленту после всех попыток")
}

// saveFeedData сохраняет данные из RSS-ленты в базу данных
func saveFeedData(feedURL string, rss RSS) error {
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
			logrus.WithError(err).WithFields(logrus.Fields{
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
			logrus.WithError(err).WithFields(logrus.Fields{
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
		logrus.WithError(err).Error("Ошибка получения общего количества записей")
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
		SELECT n.id, n.title, n.description, n.content, n.publication_date, n.source_link, s.name as source_name
		FROM news n
		JOIN rss_feeds rf ON n.rss_feed_id = rf.id
		JOIN sources s ON rf.source_id = s.id
		WHERE n.id = $1
	`, newsID).Scan(&news.ID, &news.Title, &news.Description, &news.Content, &news.PublicationDate, &news.SourceLink, &news.SourceName)
	if err != nil {
		logrus.WithError(err).Error("Ошибка получения деталей новости")
		http.Error(w, "Новость не найдена", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(news)
}
