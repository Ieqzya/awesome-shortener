package service

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"awesome-shortener/internal/model"
)

// FileStorage представляет файловое хранилище URL
type FileStorage struct {
	filePath string
	urls     map[string]string // shortURL -> originalURL
	records  map[string]*model.URLRecord // shortURL -> URLRecord
	counter  int
	mutex    sync.RWMutex
}

// NewFileStorage создает новое файловое хранилище
func NewFileStorage(filePath string) *FileStorage {
	fs := &FileStorage{
		filePath: filePath,
		urls:     make(map[string]string),
		records:  make(map[string]*model.URLRecord),
		counter:  0,
	}
	
	// Загружаем данные из файла при инициализации
	fs.loadFromFile()
	
	return fs
}

// Store сохраняет URL в хранилище
func (fs *FileStorage) Store(shortURL, originalURL string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	
	fs.counter++
	uuid := fmt.Sprintf("%d", fs.counter)
	
	record := &model.URLRecord{
		UUID:        uuid,
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	
	fs.urls[shortURL] = originalURL
	fs.records[shortURL] = record
	
	return fs.saveToFile()
}

// Get получает оригинальный URL по короткому
func (fs *FileStorage) Get(shortURL string) (string, bool) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	
	originalURL, exists := fs.urls[shortURL]
	return originalURL, exists
}

// loadFromFile загружает данные из файла
func (fs *FileStorage) loadFromFile() error {
	if _, err := os.Stat(fs.filePath); os.IsNotExist(err) {
		// Файл не существует, это нормально для первого запуска
		return nil
	}
	
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %w", err)
	}
	
	if len(data) == 0 {
		// Пустой файл
		return nil
	}
	
	var records []model.URLRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %w", err)
	}
	
	// Восстанавливаем данные в память
	maxUUID := 0
	for _, record := range records {
		fs.urls[record.ShortURL] = record.OriginalURL
		fs.records[record.ShortURL] = &record
		
		// Находим максимальный UUID для продолжения счетчика
		var uuid int
		if _, err := fmt.Sscanf(record.UUID, "%d", &uuid); err == nil {
			if uuid > maxUUID {
				maxUUID = uuid
			}
		}
	}
	
	fs.counter = maxUUID
	return nil
}

// saveToFile сохраняет данные в файл
func (fs *FileStorage) saveToFile() error {
	records := make([]model.URLRecord, 0, len(fs.records))
	for _, record := range fs.records {
		records = append(records, *record)
	}
	
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}
	
	// Создаем директорию если не существует
	if err := os.MkdirAll(fs.filePath[:len(fs.filePath)-len("/short-url-db.json")], 0755); err != nil {
		// Если не удалось создать директорию, попробуем записать в текущую
	}
	
	if err := os.WriteFile(fs.filePath, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}
	
	return nil
}