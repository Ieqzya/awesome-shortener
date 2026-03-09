package storage

import (
	"context"
	"fmt"
	"sync"
)

// MemoryURLRecord представляет внутреннюю структуру для хранения URL в памяти.
//
// Содержит дополнительные поля для поддержки функциональности
// пользователей и мягкого удаления.
type MemoryURLRecord struct {
	OriginalURL string // оригинальный URL
	UserID      string // идентификатор пользователя-владельца
	IsDeleted   bool   // флаг мягкого удаления
}

// MemoryStorage реализует хранилище URL в оперативной памяти.
//
// Предоставляет быстрое хранилище для разработки и тестирования.
// Все данные теряются при перезапуске приложения.
// Реализация потокобезопасна.
type MemoryStorage struct {
	urls        map[string]MemoryURLRecord // карта для хранения URL записей
	urlToShortID map[string]string         // обратный индекс: originalURL -> shortID для O(1) поиска
	mu          sync.RWMutex               // мьютекс для потокобезопасности
}

// NewMemoryStorage создает новое хранилище в оперативной памяти.
//
// Возвращает инициализированный MemoryStorage с пустой картой URL.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls:        make(map[string]MemoryURLRecord),
		urlToShortID: make(map[string]string),
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

	// Проверяем конфликт через обратный индекс O(1)
	if existingShortID, exists := m.urlToShortID[originalURL]; exists {
		return &ErrConflict{ShortID: existingShortID}
	}

	m.urls[shortID] = MemoryURLRecord{
		OriginalURL: originalURL,
		UserID:      userID,
	}
	m.urlToShortID[originalURL] = shortID
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

	// Используем обратный индекс для O(1) поиска
	shortID, exists := m.urlToShortID[originalURL]
	if !exists {
		return "", fmt.Errorf("URL не найден")
	}
	return shortID, nil
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
		m.urlToShortID[item.OriginalURL] = item.ShortID
	}
	return nil
}

// GetStats возвращает статистику: количество URL и пользователей
func (m *MemoryStorage) GetStats(ctx context.Context) (int, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	urlsCount := len(m.urls)

	// Подсчитываем уникальных пользователей
	users := make(map[string]struct{})
	for _, record := range m.urls {
		if record.UserID != "" {
			users[record.UserID] = struct{}{}
		}
	}
	usersCount := len(users)

	return urlsCount, usersCount, nil
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
			// Удаляем из обратного индекса при мягком удалении
			delete(m.urlToShortID, record.OriginalURL)
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
