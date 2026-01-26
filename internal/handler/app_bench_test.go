package handler

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/storage"
)

// BenchmarkCreateShortURL бенчмарк для создания коротких URL
func BenchmarkCreateShortURL(b *testing.B) {
	cfg := &config.Config{
		ServerAddress:   "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "/tmp/bench-test.json",
	}
	
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)
	defer app.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		body := strings.NewReader("https://example.com/test")
		req := httptest.NewRequest("POST", "/", body)
		w := httptest.NewRecorder()

		app.CreateShortURL(w, req)
	}
}

// BenchmarkCreateShortURLJSON бенчмарк для создания коротких URL через JSON API
func BenchmarkCreateShortURLJSON(b *testing.B) {
	cfg := &config.Config{
		ServerAddress:   "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "/tmp/bench-test.json",
	}
	
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)
	defer app.Shutdown()

	jsonBody := `{"url":"https://example.com/test"}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/shorten", bytes.NewBufferString(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.CreateShortURLJSON(w, req)
	}
}

// BenchmarkRedirectToOriginal бенчмарк для перенаправления
func BenchmarkRedirectToOriginal(b *testing.B) {
	cfg := &config.Config{
		ServerAddress:   "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "/tmp/bench-test.json",
	}
	
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)
	defer app.Shutdown()

	// Предварительно сохраняем URL
	ctx := context.Background()
	store.SaveURLWithUser(ctx, "testid", "https://example.com/test", "user123")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/testid", nil)
		w := httptest.NewRecorder()

		app.RedirectToOriginal(w, req)
	}
}

// BenchmarkBatchCreate бенчмарк для batch создания URL
func BenchmarkBatchCreate(b *testing.B) {
	cfg := &config.Config{
		ServerAddress:   "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "/tmp/bench-test.json",
	}
	
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)
	defer app.Shutdown()

	jsonBody := `[
		{"correlation_id":"1","original_url":"https://example1.com"},
		{"correlation_id":"2","original_url":"https://example2.com"},
		{"correlation_id":"3","original_url":"https://example3.com"}
	]`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/shorten/batch", bytes.NewBufferString(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.CreateShortURLBatch(w, req)
	}
}