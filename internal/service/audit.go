package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"awesome-shortener/internal/model"
)

// AuditObserver интерфейс наблюдателя для аудита
type AuditObserver interface {
	Notify(event model.AuditEvent) error
}

// AuditSubject интерфейс субъекта для аудита
type AuditSubject interface {
	Subscribe(observer AuditObserver)
	Unsubscribe(observer AuditObserver)
	NotifyAll(event model.AuditEvent)
}

// AuditService реализует паттерн Subject для аудита
type AuditService struct {
	observers []AuditObserver
	mutex     sync.RWMutex
}

// NewAuditService создает новый сервис аудита
func NewAuditService() *AuditService {
	return &AuditService{
		observers: make([]AuditObserver, 0),
	}
}

// Subscribe добавляет наблюдателя
func (as *AuditService) Subscribe(observer AuditObserver) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	as.observers = append(as.observers, observer)
}

// Unsubscribe удаляет наблюдателя
func (as *AuditService) Unsubscribe(observer AuditObserver) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	for i, obs := range as.observers {
		if obs == observer {
			as.observers = append(as.observers[:i], as.observers[i+1:]...)
			break
		}
	}
}

// NotifyAll уведомляет всех наблюдателей
func (as *AuditService) NotifyAll(event model.AuditEvent) {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	for _, observer := range as.observers {
		// Запускаем в горутине, чтобы не блокировать основной поток
		go func(obs AuditObserver) {
			if err := obs.Notify(event); err != nil {
				// В реальном приложении здесь должно быть логирование
				fmt.Printf("Ошибка аудита: %v\n", err)
			}
		}(observer)
	}
}

// FileAuditObserver реализует запись аудита в файл
type FileAuditObserver struct {
	filePath string
	mutex    sync.Mutex
}

// NewFileAuditObserver создает наблюдателя для записи в файл
func NewFileAuditObserver(filePath string) *FileAuditObserver {
	return &FileAuditObserver{
		filePath: filePath,
	}
}

// Notify записывает событие в файл
func (fao *FileAuditObserver) Notify(event model.AuditEvent) error {
	fao.mutex.Lock()
	defer fao.mutex.Unlock()

	// Создаем директорию если не существует
	dir := filepath.Dir(fao.filePath)
	if dir != "." && dir != "" {
		os.MkdirAll(dir, 0755)
	}

	// Открываем файл для записи в append режиме
	file, err := os.OpenFile(fao.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла аудита: %w", err)
	}
	defer file.Close()

	// Сериализуем событие в JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("ошибка сериализации события аудита: %w", err)
	}

	// Добавляем перенос строки
	data = append(data, '\n')

	// Записываем в файл
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("ошибка записи в файл аудита: %w", err)
	}

	return file.Sync()
}

// HTTPAuditObserver реализует отправку аудита на удаленный сервер
type HTTPAuditObserver struct {
	url    string
	client *http.Client
}

// NewHTTPAuditObserver создает наблюдателя для отправки по HTTP
func NewHTTPAuditObserver(url string) *HTTPAuditObserver {
	return &HTTPAuditObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify отправляет событие на удаленный сервер
func (hao *HTTPAuditObserver) Notify(event model.AuditEvent) error {
	// Сериализуем событие в JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("ошибка сериализации события аудита: %w", err)
	}

	// Создаем POST запрос
	req, err := http.NewRequest("POST", hao.url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	resp, err := hao.client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	return nil
}
