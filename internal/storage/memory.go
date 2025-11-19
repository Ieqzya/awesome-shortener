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

// Close закрывает хранилище (для памяти ничего не делает)
func (m *MemoryStorage) Close() error {
	return nil
}
