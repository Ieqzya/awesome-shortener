package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"awesome-shortener/internal/config"
	"github.com/go-chi/chi/v5"
)

func init() {
	// Инициализируем конфигурацию для тестов
	cfg = &config.Config{
		ServerAddress: config.DefaultServerAddress,
		BaseURL:       config.DefaultBaseURL,
	}
}

func TestCreateShortURL(t *testing.T) {
	urls = make(map[string]string)

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

	if len(urls) != 1 {
		t.Errorf("URL не сохранился")
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
	urls = make(map[string]string)
	urls["test123"] = "https://example.com"

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
	urls = make(map[string]string)

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
	urls = make(map[string]string)

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

	if len(urls) != 1 {
		t.Errorf("URL не сохранился")
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