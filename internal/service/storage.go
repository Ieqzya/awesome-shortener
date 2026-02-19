package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"awesome-shortener/internal/model"
	"awesome-shortener/internal/repository"
)

// Убеждаемся, что FileStorage реализует интерфейс Storage
var _ repository.Storage = (*FileStorage)(nil)

// FileStorage представляет файловое хранилище URL
type FileStorage struct {
	filePath string
	urls     map[string]string // shortURL -> originalURL
	counter  int
	mutex    sync.RWMutex
	file     *os.File // файл для записи в append режиме
}

// NewFileStorage создает новое файловое хранилище
func NewFileStorage(filePath string) *FileStorage {
	fs := &FileStorage{
		filePath: filePath,
		urls:     make(map[string]string),
		counter:  0,
	}

	// Загружаем данные из файла при инициализации
	fs.loadFromFile()

	// Открываем файл для записи в append режиме
	fs.openFileForAppend()

	return fs
}

// openFileForAppend открывает файл для записи в append режиме
func (fs *FileStorage) openFileForAppend() error {
	// Создаем директорию если не существует
	dir := filepath.Dir(fs.filePath)
	if dir != "." && dir != "" {
		os.MkdirAll(dir, 0755)
	}

	file, err := os.OpenFile(fs.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	fs.file = file
	return nil
}

// Close закрывает файл
func (fs *FileStorage) Close() error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.file != nil {
		return fs.file.Close()
	}
	return nil
}

// Store сохраняет URL в хранилище
func (fs *FileStorage) Store(shortURL, originalURL string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	fs.counter++
	uuid := fmt.Sprintf("%d", fs.counter)

	record := model.URLRecord{
		UUID:        uuid,
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}

	// Сохраняем в память
	fs.urls[shortURL] = originalURL

	// Записываем в файл в NDJSON формате (одна строка JSON)
	return fs.appendToFile(record)
}

// Get получает оригинальный URL по короткому
func (fs *FileStorage) Get(shortURL string) (string, bool) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	originalURL, exists := fs.urls[shortURL]
	return originalURL, exists
}

// loadFromFile загружает данные из файла в NDJSON формате
func (fs *FileStorage) loadFromFile() error {
	if _, err := os.Stat(fs.filePath); os.IsNotExist(err) {
		// Файл не существует, это нормально для первого запуска
		return nil
	}

	file, err := os.Open(fs.filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	maxUUID := 0

	// Читаем файл построчно (NDJSON)
	for decoder.More() {
		var record model.URLRecord
		if err := decoder.Decode(&record); err != nil {
			// Пропускаем поврежденные строки
			continue
		}

		// Восстанавливаем данные в память
		fs.urls[record.ShortURL] = record.OriginalURL

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

// appendToFile добавляет новую запись в конец файла в NDJSON формате
func (fs *FileStorage) appendToFile(record model.URLRecord) error {
	if fs.file == nil {
		if err := fs.openFileForAppend(); err != nil {
			return fmt.Errorf("ошибка открытия файла: %w", err)
		}
	}

	// Сериализуем запись в JSON
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	// Добавляем перенос строки для NDJSON формата
	data = append(data, '\n')

	// Записываем в файл
	if _, err := fs.file.Write(data); err != nil {
		return fmt.Errorf("ошибка записи в файл: %w", err)
	}

	// Принудительно сбрасываем буфер на диск
	return fs.file.Sync()
}
