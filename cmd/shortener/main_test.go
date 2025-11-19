package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/handler"
	"awesome-shortener/internal/middleware"
	"awesome-shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func TestCreateShortURL(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: config.DefaultServerAddress,
		BaseURL:       config.DefaultBaseURL,
	}
	store := storage.NewMemoryStorage()
	app := handler.NewApp(cfg, store)

	body := strings.NewReader("https://google.com")
	req := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()

	app.CreateShortURL(w, req)

	if w.Code != 201 {
		t.Errorf("Ожидали код 201, получили %d", w.Code)
	}

	response := w.Body.String()
	if !strings.Contains(response, "http://localhost:8080/") {
		t.Errorf("Неправильный ответ: %s", response)
	}
}

func TestCreateShortURLBadRequest(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: config.DefaultServerAddress,
		BaseURL:       config.DefaultBaseURL,
	}
	store := storage.NewMemoryStorage()
	app := handler.NewApp(cfg, store)

	req := httptest.NewRequest("POST", "/", strings.NewReader(""))
	w := httptest.NewRecorder()

	app.CreateShortURL(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400, получили %d", w.Code)
	}
}

func TestRedirectToOriginal(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: config.DefaultServerAddress,
		BaseURL:       config.DefaultBaseURL,
	}
	store := storage.NewMemoryStorage()
	store.SaveURL(context.Background(), "test123", "https://example.com")
	app := handler.NewApp(cfg, store)

	r := chi.NewRouter()
	r.Get("/{id}", app.RedirectToOriginal)

	req := httptest.NewRequest("GET", "/test123", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != 307 {
		t.Errorf("Ожидали код 307, получили %d", w.Code)
	}

	if w.Header().Get("Location") != "https://example.com" {
		t.Errorf("Неправильный Location")
	}
}

func TestRedirectNotFound(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: config.DefaultServerAddress,
		BaseURL:       config.DefaultBaseURL,
	}
	store := storage.NewMemoryStorage()
	app := handler.NewApp(cfg, store)

	r := chi.NewRouter()
	r.Get("/{id}", app.RedirectToOriginal)

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400, получили %d", w.Code)
	}
}

func TestCreateShortURLJSON(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: config.DefaultServerAddress,
		BaseURL:       config.DefaultBaseURL,
	}
	store := storage.NewMemoryStorage()
	app := handler.NewApp(cfg, store)

	jsonBody := `{"url":"https://practicum.yandex.ru"}`
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.CreateShortURLJSON(w, req)

	if w.Code != 201 {
		t.Errorf("Ожидали код 201, получили %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Ожидали Content-Type: application/json, получили %s", w.Header().Get("Content-Type"))
	}

	var response handler.ShortenResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Ошибка декодирования JSON ответа: %v", err)
	}

	if !strings.Contains(response.Result, "http://localhost:8080/") {
		t.Errorf("Неправильный ответ: %s", response.Result)
	}
}

func TestCreateShortURLJSONBadRequest(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: config.DefaultServerAddress,
		BaseURL:       config.DefaultBaseURL,
	}
	store := storage.NewMemoryStorage()
	app := handler.NewApp(cfg, store)

	// Тест с невалидным JSON
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"invalid": json}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.CreateShortURLJSON(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400, получили %d", w.Code)
	}
}

func TestCreateShortURLJSONEmptyURL(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: config.DefaultServerAddress,
		BaseURL:       config.DefaultBaseURL,
	}
	store := storage.NewMemoryStorage()
	app := handler.NewApp(cfg, store)

	// Тест с пустым URL
	jsonBody := `{"url":""}`
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.CreateShortURLJSON(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400, получили %d", w.Code)
	}
}

func TestGzipCompression(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: config.DefaultServerAddress,
		BaseURL:       config.DefaultBaseURL,
	}
	store := storage.NewMemoryStorage()
	app := handler.NewApp(cfg, store)

	// Создаем роутер с middleware
	logger, _ := zap.NewProduction()
	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.Logger(logger))
	r.Post("/api/shorten", app.CreateShortURLJSON)

	jsonBody := `{"url":"https://practicum.yandex.ru"}`
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("Ожидали код 201, получили %d", w.Code)
	}

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Ожидали Content-Encoding: gzip, получили %s", w.Header().Get("Content-Encoding"))
	}

	// Проверяем, что ответ действительно сжат
	reader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Errorf("Ошибка создания gzip reader: %v", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Ошибка чтения сжатых данных: %v", err)
	}

	var response handler.ShortenResponse
	if err := json.Unmarshal(decompressed, &response); err != nil {
		t.Errorf("Ошибка декодирования JSON ответа: %v", err)
	}

	if !strings.Contains(response.Result, "http://localhost:8080/") {
		t.Errorf("Неправильный ответ: %s", response.Result)
	}
}

func TestGzipDecompression(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: config.DefaultServerAddress,
		BaseURL:       config.DefaultBaseURL,
	}
	store := storage.NewMemoryStorage()
	app := handler.NewApp(cfg, store)

	// Создаем роутер с middleware
	logger, _ := zap.NewProduction()
	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.Logger(logger))
	r.Post("/api/shorten", app.CreateShortURLJSON)

	// Сжимаем тело запроса
	jsonBody := `{"url":"https://practicum.yandex.ru"}`
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Write([]byte(jsonBody))
	gzWriter.Close()

	req := httptest.NewRequest("POST", "/api/shorten", &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("Ожидали код 201, получили %d", w.Code)
	}

	var response handler.ShortenResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Ошибка декодирования JSON ответа: %v", err)
	}

	if !strings.Contains(response.Result, "http://localhost:8080/") {
		t.Errorf("Неправильный ответ: %s", response.Result)
	}
}

func TestCreateShortURLBatchIntegration(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: config.DefaultServerAddress,
		BaseURL:       config.DefaultBaseURL,
	}
	store := storage.NewMemoryStorage()
	app := handler.NewApp(cfg, store)

	// Создаем роутер с middleware
	logger, _ := zap.NewProduction()
	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.Logger(logger))
	r.Post("/api/shorten/batch", app.CreateShortURLBatch)

	// Подготавливаем batch запрос
	requests := []handler.BatchShortenRequest{
		{
			CorrelationID: "test-id-1",
			OriginalURL:   "https://example1.com",
		},
		{
			CorrelationID: "test-id-2",
			OriginalURL:   "https://example2.com",
		},
	}

	body, _ := json.Marshal(requests)
	req := httptest.NewRequest("POST", "/api/shorten/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("Ожидали код 201, получили %d", w.Code)
	}

	var responses []handler.BatchShortenResponse
	if err := json.NewDecoder(w.Body).Decode(&responses); err != nil {
		t.Fatalf("Ошибка декодирования ответа: %v", err)
	}

	if len(responses) != 2 {
		t.Errorf("Ожидали 2 элемента в ответе, получили %d", len(responses))
	}

	// Проверяем correlation_id и short_url
	for i, resp := range responses {
		if resp.CorrelationID != requests[i].CorrelationID {
			t.Errorf("Неправильный correlation_id: ожидали %s, получили %s",
				requests[i].CorrelationID, resp.CorrelationID)
		}
		if !strings.Contains(resp.ShortURL, "http://localhost:8080/") {
			t.Errorf("Неправильный short_url: %s", resp.ShortURL)
		}
	}
}
