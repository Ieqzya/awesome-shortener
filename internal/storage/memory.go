package storage

import (
	"context"
	"fmt"
	"sync"
)

// MemoryURLRecord внутренняя структура для хранения URL с user_id
type MemoryURLRecord struct {
	OriginalURL string
	UserID      string
	IsDeleted   bool
}

// MemoryStorage хранилище URL в памяти
type MemoryStorage struct {
	urls map[string]MemoryURLRecord
	mu   sync.RWMutex
}

// NewMemoryStorage создает новое хранилище в памяти
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls: make(map[string]MemoryURLRecord),
	}
}

// SaveURL сохраняет URL в памяти без user_id
func (m *MemoryStorage) SaveURL(ctx context.Context, shortID, originalURL string) error {
	return m.SaveURLWithUser(ctx, shortID, originalURL, "")
}

// SaveURLWithUser сохраняет URL в памяти с user_id
func (m *MemoryStorage) SaveURLWithUser(ctx context.Context, shortID, originalURL, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Проверяем, существует ли уже такой URL
	for existingShortID, record := range m.urls {
		if record.OriginalURL == originalURL {
			return &ErrConflict{ShortID: existingShortID}
		}
	}

	m.urls[shortID] = MemoryURLRecord{
		OriginalURL: originalURL,
		UserID:      userID,
	}
	return nil
}

// GetURL получает оригинальный URL по короткому ID
func (m *MemoryStorage) GetURL(ctx context.Context, shortID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, exists := m.urls[shortID]
	if !exists {
		return "", fmt.Errorf("URL не найден")
	}
	if record.IsDeleted {
		return "", &ErrDeleted{}
	}
	return record.OriginalURL, nil
}

// GetByOriginalURL получает короткий ID по оригинальному URL
func (m *MemoryStorage) GetByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for shortID, record := range m.urls {
		if record.OriginalURL == originalURL {
			return shortID, nil
		}
	}
	return "", fmt.Errorf("URL не найден")
}

// GetUserURLs получает все URL пользователя
func (m *MemoryStorage) GetUserURLs(ctx context.Context, userID string) ([]UserURLRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var records []UserURLRecord
	for shortID, record := range m.urls {
		if record.UserID == userID {
			records = append(records, UserURLRecord{
				ShortURL:    shortID,
				OriginalURL: record.OriginalURL,
			})
		}
	}

	return records, nil
}

// SaveBatch сохраняет множество URL в памяти без user_id
func (m *MemoryStorage) SaveBatch(ctx context.Context, items []BatchItem) error {
	return m.SaveBatchWithUser(ctx, items, "")
}

// SaveBatchWithUser сохраняет множество URL в памяти с user_id
func (m *MemoryStorage) SaveBatchWithUser(ctx context.Context, items []BatchItem, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, item := range items {
		m.urls[item.ShortID] = MemoryURLRecord{
			OriginalURL: item.OriginalURL,
			UserID:      userID,
		}
	}
	return nil
}

// Close закрывает хранилище (для памяти ничего не делает)
func (m *MemoryStorage) Close() error {
	return nil
}

// DeleteURLs помечает URL как удаленные
func (m *MemoryStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, shortID := range shortIDs {
		if record, exists := m.urls[shortID]; exists && record.UserID == userID {
			record.IsDeleted = true
			m.urls[shortID] = record
		}
	}
	return nil
}

// IsDeleted проверяет, удален ли URL
func (m *MemoryStorage) IsDeleted(ctx context.Context, shortID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, exists := m.urls[shortID]
	if !exists {
		return false, fmt.Errorf("URL не найден")
	}
	return record.IsDeleted, nil
}
