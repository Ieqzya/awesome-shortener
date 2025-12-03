package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/storage"
)

func TestCreateShortURLConflict(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)

	// Первый запрос - создаем URL
	body1 := strings.NewReader("https://example.com")
	req1 := httptest.NewRequest("POST", "/", body1)
	w1 := httptest.NewRecorder()

	app.CreateShortURL(w1, req1)

	if w1.Code != 201 {
		t.Errorf("Первый запрос: ожидали код 201, получили %d", w1.Code)
	}

	firstShortURL := w1.Body.String()

	// Второй запрос - тот же URL, должен вернуть 409
	body2 := strings.NewReader("https://example.com")
	req2 := httptest.NewRequest("POST", "/", body2)
	w2 := httptest.NewRecorder()

	app.CreateShortURL(w2, req2)

	if w2.Code != 409 {
		t.Errorf("Второй запрос: ожидали код 409, получили %d", w2.Code)
	}

	secondShortURL := w2.Body.String()

	// Проверяем, что вернулся тот же short URL
	if firstShortURL != secondShortURL {
		t.Errorf("Short URL должны совпадать: первый=%s, второй=%s", firstShortURL, secondShortURL)
	}
}

func TestCreateShortURLJSONConflict(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)

	// Первый запрос - создаем URL
	jsonBody1 := `{"url":"https://practicum.yandex.ru"}`
	req1 := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(jsonBody1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()

	app.CreateShortURLJSON(w1, req1)

	if w1.Code != 201 {
		t.Errorf("Первый запрос: ожидали код 201, получили %d", w1.Code)
	}

	var response1 ShortenResponse
	json.NewDecoder(w1.Body).Decode(&response1)

	// Второй запрос - тот же URL, должен вернуть 409
	jsonBody2 := `{"url":"https://practicum.yandex.ru"}`
	req2 := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(jsonBody2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	app.CreateShortURLJSON(w2, req2)

	if w2.Code != 409 {
		t.Errorf("Второй запрос: ожидали код 409, получили %d", w2.Code)
	}

	var response2 ShortenResponse
	json.NewDecoder(w2.Body).Decode(&response2)

	// Проверяем, что вернулся тот же short URL
	if response1.Result != response2.Result {
		t.Errorf("Short URL должны совпадать: первый=%s, второй=%s", response1.Result, response2.Result)
	}
}
