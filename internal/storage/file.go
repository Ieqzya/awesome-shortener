package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// FileStorage хранилище URL в файле
type FileStorage struct {
	filePath string
	urls     map[string]string
	file     *os.File
	mu       sync.RWMutex
}

// URLRecord запись URL в файле
type URLRecord struct {
	ShortID     string `json:"short_id"`
	OriginalURL string `json:"original_url"`
}

// NewFileStorage создает новое файловое хранилище
func NewFileStorage(filePath string) (*FileStorage, error) {
	if filePath == "" {
		return nil, nil
	}

	storage := &FileStorage{
		filePath: filePath,
		urls:     make(map[string]string),
	}

	// Открываем файл для чтения и записи
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла: %w", err)
	}
	storage.file = file

	// Загружаем существующие данные
	if err := storage.loadFromFile(); err != nil {
		file.Close()
		return nil, fmt.Errorf("ошибка загрузки данных: %w", err)
	}

	return storage, nil
}

// loadFromFile загружает данные из файла
func (f *FileStorage) loadFromFile() error {
	scanner := bufio.NewScanner(f.file)
	for scanner.Scan() {
		var record URLRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue // Пропускаем некорректные записи
		}
		f.urls[record.ShortID] = record.OriginalURL
	}
	return scanner.Err()
}

// SaveURL сохраняет URL в файл
func (f *FileStorage) SaveURL(ctx context.Context, shortID, originalURL string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Проверяем, существует ли уже такой URL
	for existingShortID, url := range f.urls {
		if url == originalURL {
			return &ErrConflict{ShortID: existingShortID}
		}
	}

	// Сохраняем в памяти
	f.urls[shortID] = originalURL

	// Записываем в файл
	record := URLRecord{
		ShortID:     shortID,
		OriginalURL: originalURL,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %w", err)
	}

	if _, err := f.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("ошибка записи в файл: %w", err)
	}

	return nil
}

// GetURL получает оригинальный URL по короткому ID
func (f *FileStorage) GetURL(ctx context.Context, shortID string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	url, exists := f.urls[shortID]
	if !exists {
		return "", fmt.Errorf("URL не найден")
	}
	return url, nil
}

// GetByOriginalURL получает короткий ID по оригинальному URL
func (f *FileStorage) GetByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	for shortID, url := range f.urls {
		if url == originalURL {
			return shortID, nil
		}
	}
	return "", fmt.Errorf("URL не найден")
}

// SaveURLWithUser сохраняет URL в файл с user_id (для совместимости)
func (f *FileStorage) SaveURLWithUser(ctx context.Context, shortID, originalURL, userID string) error {
	return f.SaveURL(ctx, shortID, originalURL)
}

// GetUserURLs получает все URL пользователя (не поддерживается для файлового хранилища)
func (f *FileStorage) GetUserURLs(ctx context.Context, userID string) ([]URLRecord, error) {
	return []URLRecord{}, nil
}

// SaveBatch сохраняет множество URL в файл без user_id
func (f *FileStorage) SaveBatch(ctx context.Context, items []BatchItem) error {
	return f.SaveBatchWithUser(ctx, items, "")
}

// SaveBatchWithUser сохраняет множество URL в файл с user_id
func (f *FileStorage) SaveBatchWithUser(ctx context.Context, items []BatchItem, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, item := range items {
		// Сохраняем в памяти
		f.urls[item.ShortID] = item.OriginalURL

		// Записываем в файл
		record := URLRecord{
			ShortID:     item.ShortID,
			OriginalURL: item.OriginalURL,
		}

		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("ошибка сериализации: %w", err)
		}

		if _, err := f.file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("ошибка записи в файл: %w", err)
		}
	}

	return nil
}

// Close закрывает файл
func (f *FileStorage) Close() error {
	if f.file != nil {
		return f.file.Close()
	}
	return nil
}
