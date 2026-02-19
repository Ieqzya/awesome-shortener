package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/storage"
)

func TestCreateShortURLBatch(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)

	// Подготавливаем batch запрос
	requests := []BatchShortenRequest{
		{
			CorrelationID: "id1",
			OriginalURL:   "https://example1.com",
		},
		{
			CorrelationID: "id2",
			OriginalURL:   "https://example2.com",
		},
		{
			CorrelationID: "id3",
			OriginalURL:   "https://example3.com",
		},
	}

	body, _ := json.Marshal(requests)
	req := httptest.NewRequest("POST", "/api/shorten/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.CreateShortURLBatch(w, req)

	if w.Code != 201 {
		t.Errorf("Ожидали код 201, получили %d", w.Code)
	}

	var responses []BatchShortenResponse
	if err := json.NewDecoder(w.Body).Decode(&responses); err != nil {
		t.Fatalf("Ошибка декодирования ответа: %v", err)
	}

	if len(responses) != 3 {
		t.Errorf("Ожидали 3 элемента в ответе, получили %d", len(responses))
	}

	// Проверяем correlation_id
	for i, resp := range responses {
		if resp.CorrelationID != requests[i].CorrelationID {
			t.Errorf("Неправильный correlation_id: ожидали %s, получили %s",
				requests[i].CorrelationID, resp.CorrelationID)
		}
		if resp.ShortURL == "" {
			t.Error("ShortURL не должен быть пустым")
		}
	}
}

func TestCreateShortURLBatchEmpty(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)

	// Пустой batch
	requests := []BatchShortenRequest{}
	body, _ := json.Marshal(requests)
	req := httptest.NewRequest("POST", "/api/shorten/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.CreateShortURLBatch(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400 для пустого batch, получили %d", w.Code)
	}
}

func TestCreateShortURLBatchInvalidJSON(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)

	req := httptest.NewRequest("POST", "/api/shorten/batch", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.CreateShortURLBatch(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400 для невалидного JSON, получили %d", w.Code)
	}
}

func TestCreateShortURLBatchEmptyURL(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)

	// Batch с пустым URL
	requests := []BatchShortenRequest{
		{
			CorrelationID: "id1",
			OriginalURL:   "",
		},
	}
	body, _ := json.Marshal(requests)
	req := httptest.NewRequest("POST", "/api/shorten/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.CreateShortURLBatch(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400 для пустого URL, получили %d", w.Code)
	}
}
