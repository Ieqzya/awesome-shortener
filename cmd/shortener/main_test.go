package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/middleware"
	"awesome-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func init() {
	// Инициализируем конфигурацию для тестов
	cfg = &config.Config{
		ServerAddress:   config.DefaultServerAddress,
		BaseURL:         config.DefaultBaseURL,
		FileStoragePath: "/tmp/test-short-url-db.json",
	}
}

func TestCreateShortURL(t *testing.T) {
	storage = service.NewFileStorage("/tmp/test-short-url-db-1.json")

	body := strings.NewReader("https://google.com")
	req := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()

	createShortURL(w, req)

	if w.Code != 201 {
		t.Errorf("Ожидали код 201, получили %d", w.Code)
	}

	response := w.Body.String()
	if !strings.Contains(response, "http://localhost:8080/") {
		t.Errorf("Неправильный ответ: %s", response)
	}
}

func TestCreateShortURLBadRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(""))
	w := httptest.NewRecorder()

	createShortURL(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400, получили %d", w.Code)
	}
}

func TestRedirectToOriginal(t *testing.T) {
	storage = service.NewFileStorage("/tmp/test-short-url-db-2.json")
	storage.Store("test123", "https://example.com")

	r := chi.NewRouter()
	r.Get("/{id}", redirectToOriginal)

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
	storage = service.NewFileStorage("/tmp/test-short-url-db-3.json")

	r := chi.NewRouter()
	r.Get("/{id}", redirectToOriginal)

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400, получили %d", w.Code)
	}
}

func TestCreateShortURLJSON(t *testing.T) {
	storage = service.NewFileStorage("/tmp/test-short-url-db-4.json")

	jsonBody := `{"url":"https://practicum.yandex.ru"}`
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createShortURLJSON(w, req)

	if w.Code != 201 {
		t.Errorf("Ожидали код 201, получили %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Ожидали Content-Type: application/json, получили %s", w.Header().Get("Content-Type"))
	}

	var response ShortenResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Ошибка декодирования JSON ответа: %v", err)
	}

	if !strings.Contains(response.Result, "http://localhost:8080/") {
		t.Errorf("Неправильный ответ: %s", response.Result)
	}
}

func TestCreateShortURLJSONBadRequest(t *testing.T) {
	// Тест с невалидным JSON
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(`{"invalid": json}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createShortURLJSON(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400, получили %d", w.Code)
	}
}

func TestCreateShortURLJSONEmptyURL(t *testing.T) {
	// Тест с пустым URL
	jsonBody := `{"url":""}`
	req := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createShortURLJSON(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400, получили %d", w.Code)
	}
}

func TestGzipCompression(t *testing.T) {
	storage = service.NewFileStorage("/tmp/test-short-url-db-5.json")

	// Создаем роутер с middleware
	logger, _ := zap.NewProduction()
	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.Logger(logger))
	r.Post("/api/shorten", createShortURLJSON)

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

	var response ShortenResponse
	if err := json.Unmarshal(decompressed, &response); err != nil {
		t.Errorf("Ошибка декодирования JSON ответа: %v", err)
	}

	if !strings.Contains(response.Result, "http://localhost:8080/") {
		t.Errorf("Неправильный ответ: %s", response.Result)
	}
}

func TestGzipDecompression(t *testing.T) {
	storage = service.NewFileStorage("/tmp/test-short-url-db-6.json")

	// Создаем роутер с middleware
	logger, _ := zap.NewProduction()
	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.Logger(logger))
	r.Post("/api/shorten", createShortURLJSON)

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

	var response ShortenResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Ошибка декодирования JSON ответа: %v", err)
	}

	if !strings.Contains(response.Result, "http://localhost:8080/") {
		t.Errorf("Неправильный ответ: %s", response.Result)
	}
}