package storage

import (
	"context"
	"fmt"
	"sync"
)

// MemoryStorage хранилище URL в памяти
type MemoryStorage struct {
	urls map[string]string
	mu   sync.RWMutex
}

// NewMemoryStorage создает новое хранилище в памяти
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls: make(map[string]string),
	}
}

// SaveURL сохраняет URL в памяти
func (m *MemoryStorage) SaveURL(ctx context.Context, shortID, originalURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Проверяем, существует ли уже такой URL
	for existingShortID, url := range m.urls {
		if url == originalURL {
			return &ErrConflict{ShortID: existingShortID}
		}
	}
	
	m.urls[shortID] = originalURL
	return nil
}

// GetURL получает оригинальный URL по короткому ID
func (m *MemoryStorage) GetURL(ctx context.Context, shortID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	url, exists := m.urls[shortID]
	if !exists {
		return "", fmt.Errorf("URL не найден")
	}
	return url, nil
}

// GetByOriginalURL получает короткий ID по оригинальному URL
func (m *MemoryStorage) GetByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for shortID, url := range m.urls {
		if url == originalURL {
			return shortID, nil
		}
	}
	return "", fmt.Errorf("URL не найден")
}

// SaveBatch сохраняет множество URL в памяти
func (m *MemoryStorage) SaveBatch(ctx context.Context, items []BatchItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for _, item := range items {
		m.urls[item.ShortID] = item.OriginalURL
	}
	return nil
}

// Close закрывает хранилище (для памяти ничего не делает)
func (m *MemoryStorage) Close() error {
	return nil
}
