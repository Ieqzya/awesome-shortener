package storage

import (
	"context"
	"fmt"
	"testing"
)

// BenchmarkMemoryStorageSave бенчмарк для сохранения в память
func BenchmarkMemoryStorageSave(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		store.SaveURLWithUser(ctx, generateID(i), "https://example.com/test", "user123")
	}
}

// BenchmarkMemoryStorageGet бенчмарк для получения из памяти
func BenchmarkMemoryStorageGet(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()

	// Предварительно сохраняем данные
	for i := 0; i < 1000; i++ {
		store.SaveURLWithUser(ctx, generateID(i), "https://example.com/test", "user123")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		store.GetURL(ctx, generateID(i%1000))
	}
}

// BenchmarkMemoryStorageBatch бенчмарк для batch операций
func BenchmarkMemoryStorageBatch(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()

	items := []BatchItem{
		{CorrelationID: "1", ShortID: "id1", OriginalURL: "https://example1.com"},
		{CorrelationID: "2", ShortID: "id2", OriginalURL: "https://example2.com"},
		{CorrelationID: "3", ShortID: "id3", OriginalURL: "https://example3.com"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		store.SaveBatchWithUser(ctx, items, "user123")
	}
}

// generateID генерирует ID для тестов
func generateID(i int) string {
	return fmt.Sprintf("test%d", i)
}