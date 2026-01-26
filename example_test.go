package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/handler"
	"awesome-shortener/internal/middleware"
	"awesome-shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// ExampleApp_CreateShortURL демонстрирует создание короткого URL через текстовый API.
func ExampleApp_CreateShortURL() {
	// Создаем конфигурацию
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}

	// Создаем хранилище в памяти
	store := storage.NewMemoryStorage()

	// Создаем приложение
	app := handler.NewApp(cfg, store)

	// Создаем HTTP запрос
	body := bytes.NewBufferString("https://practicum.yandex.ru")
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", "text/plain")

	// Создаем ResponseRecorder для записи ответа
	w := httptest.NewRecorder()

	// Вызываем обработчик
	app.CreateShortURL(w, req)

	// Проверяем результат
	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Content-Type: %s\n", w.Header().Get("Content-Type"))
	fmt.Printf("Body contains BaseURL: %t\n", bytes.Contains(w.Body.Bytes(), []byte(cfg.BaseURL)))

	// Output:
	// Status: 201
	// Content-Type: text/plain
	// Body contains BaseURL: true
}

// ExampleApp_CreateShortURLJSON демонстрирует создание короткого URL через JSON API.
func ExampleApp_CreateShortURLJSON() {
	// Создаем конфигурацию
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}

	// Создаем хранилище в памяти
	store := storage.NewMemoryStorage()

	// Создаем приложение
	app := handler.NewApp(cfg, store)

	// Подготавливаем JSON запрос
	requestData := handler.ShortenRequest{
		URL: "https://practicum.yandex.ru",
	}
	jsonData, _ := json.Marshal(requestData)

	// Создаем HTTP запрос
	req := httptest.NewRequest("POST", "/api/shorten", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	w := httptest.NewRecorder()

	// Вызываем обработчик
	app.CreateShortURLJSON(w, req)

	// Проверяем результат
	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Content-Type: %s\n", w.Header().Get("Content-Type"))

	// Декодируем ответ
	var response handler.ShortenResponse
	json.NewDecoder(w.Body).Decode(&response)
	fmt.Printf("Response contains result: %t\n", response.Result != "")

	// Output:
	// Status: 201
	// Content-Type: application/json
	// Response contains result: true
}

// ExampleMiddleware_Logger демонстрирует использование middleware для логирования.
func ExampleMiddleware_Logger() {
	// Создаем zap логгер для примера
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}

	// Создаем роутер с middleware
	r := chi.NewRouter()
	r.Use(middleware.Logger(logger))

	// Добавляем простой обработчик
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Создаем тестовый запрос
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Выполняем запрос
	r.ServeHTTP(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Body: %s\n", w.Body.String())

	// Output:
	// Status: 200
	// Body: OK
}

// ExampleConfig_NewConfig демонстрирует создание конфигурации.
func ExampleConfig_NewConfig() {
	// Создаем конфигурацию (в реальном приложении будут использованы флаги и переменные окружения)
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Server will start on: %s\n", cfg.ServerAddress)
	fmt.Printf("Base URL: %s\n", cfg.BaseURL)
	fmt.Printf("Config created successfully: %t\n", cfg != nil)

	// Output:
	// Server will start on: localhost:8080
	// Base URL: http://localhost:8080
	// Config created successfully: true
}

// ExampleMemoryStorage_SaveURL демонстрирует сохранение URL в памяти.
func ExampleMemoryStorage_SaveURL() {
	// Создаем хранилище в памяти
	store := storage.NewMemoryStorage()

	// Сохраняем URL
	err := store.SaveURL(nil, "abc123", "https://example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Получаем URL обратно
	originalURL, err := store.GetURL(nil, "abc123")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Saved URL: %s\n", originalURL)
	fmt.Printf("Save successful: %t\n", err == nil)

	// Output:
	// Saved URL: https://example.com
	// Save successful: true
}