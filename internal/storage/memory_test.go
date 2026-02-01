package storage

import (
	"context"
	"testing"
)

func TestMemoryStorage(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	// Тест сохранения
	err := store.SaveURL(ctx, "test123", "https://example.com")
	if err != nil {
		t.Errorf("Ошибка сохранения: %v", err)
	}

	// Тест получения
	url, err := store.GetURL(ctx, "test123")
	if err != nil {
		t.Errorf("Ошибка получения: %v", err)
	}
	if url != "https://example.com" {
		t.Errorf("Ожидали 'https://example.com', получили '%s'", url)
	}

	// Тест несуществующего URL
	_, err = store.GetURL(ctx, "notfound")
	if err == nil {
		t.Error("Ожидали ошибку для несуществующего URL")
	}
}
