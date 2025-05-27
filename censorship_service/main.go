package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"censorship_service/middleware"

	"github.com/sirupsen/logrus"
)

type CommentRequest struct {
	Text string `json:"text"`
}

func main() {
	// Настройка логгера
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	// Создаем маршрутизатор
	mux := http.NewServeMux()

	// Добавляем маршруты
	mux.HandleFunc("/api/censor", handleCensor)
	mux.HandleFunc("/health", handleHealth)

	// Добавляем middleware
	handler := middleware.LoggingMiddleware(middleware.RequestIDMiddleware(mux))

	// Запускаем сервер
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	logrus.Infof("Запуск сервиса цензуры на порту %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func handleCensor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверное тело запроса", http.StatusBadRequest)
		return
	}

	// Простая валидация: проверка на пустые или очень короткие комментарии
	if strings.TrimSpace(req.Text) == "" {
		http.Error(w, "Комментарий не может быть пустым", http.StatusBadRequest)
		return
	}

	if len(req.Text) < 3 {
		http.Error(w, "Комментарий слишком короткий", http.StatusBadRequest)
		return
	}

	// Проверка на запрещенные слова (пример)
	forbiddenWords := []string{"spam", "advertisement", "offensive"}
	for _, word := range forbiddenWords {
		if strings.Contains(strings.ToLower(req.Text), word) {
			http.Error(w, "Комментарий содержит запрещенные слова", http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}
}
