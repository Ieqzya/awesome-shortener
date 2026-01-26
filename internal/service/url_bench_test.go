package service

import (
	"context"
	"fmt"
	"testing"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/storage"
)

// BenchmarkURLServiceShortenURL бенчмарк для сокращения URL
func BenchmarkURLServiceShortenURL(b *testing.B) {
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	service := NewURLService(store, cfg)
	defer service.Shutdown()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		service.ShortenURL(ctx, "https://example.com/test", "user123")
	}
}

// BenchmarkURLServiceGetOriginalURL бенчмарк для получения оригинального URL
func BenchmarkURLServiceGetOriginalURL(b *testing.B) {
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	service := NewURLService(store, cfg)
	defer service.Shutdown()

	ctx := context.Background()

	// Предварительно сохраняем данные
	for i := 0; i < 1000; i++ {
		store.Save(ctx, generateTestID(i), "https://example.com/test", "user123")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		service.GetOriginalURL(ctx, generateTestID(i%1000))
	}
}

// BenchmarkGenerateID бенчмарк для генерации ID
func BenchmarkGenerateID(b *testing.B) {
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	service := NewURLService(store, cfg)
	defer service.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		service.GenerateID()
	}
}

func generateTestID(i int) string {
	return fmt.Sprintf("test%08d", i)
}