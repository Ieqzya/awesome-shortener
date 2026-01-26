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

// AuditObserver определяет интерфейс наблюдателя для системы аудита.
//
// Реализации этого интерфейса получают уведомления о событиях аудита
// и могут обрабатывать их различными способами (запись в файл, отправка по HTTP и т.д.).
type AuditObserver interface {
	// Notify обрабатывает событие аудита.
	// Возвращает ошибку, если обработка не удалась.
	Notify(event model.AuditEvent) error
}

// AuditSubject определяет интерфейс субъекта в паттерне Observer для аудита.
//
// Субъект управляет списком наблюдателей и уведомляет их о событиях.
type AuditSubject interface {
	// Subscribe добавляет наблюдателя в список уведомлений.
	Subscribe(observer AuditObserver)
	// Unsubscribe удаляет наблюдателя из списка уведомлений.
	Unsubscribe(observer AuditObserver)
	// NotifyAll отправляет событие всем подписанным наблюдателям.
	NotifyAll(event model.AuditEvent)
}

// AuditService реализует паттерн Subject для системы аудита.
//
// Сервис управляет списком наблюдателей и обеспечивает
// асинхронную доставку событий аудита всем подписчикам.
type AuditService struct {
	observers []AuditObserver // список наблюдателей
	mutex     sync.RWMutex    // мьютекс для безопасного доступа к списку
}

// NewAuditService создает новый экземпляр сервиса аудита.
//
// Возвращает инициализированный AuditService с пустым списком наблюдателей.
func NewAuditService() *AuditService {
	return &AuditService{
		observers: make([]AuditObserver, 0),
	}
}

// Subscribe добавляет наблюдателя в список уведомлений.
//
// Параметры:
//   - observer: наблюдатель, реализующий интерфейс AuditObserver
func (as *AuditService) Subscribe(observer AuditObserver) {
	as.mutex.Lock()
	defer as.mutex.Unlock()
	as.observers = append(as.observers, observer)
}

// Unsubscribe удаляет наблюдателя из списка уведомлений.
//
// Параметры:
//   - observer: наблюдатель для удаления
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

// NotifyAll асинхронно уведомляет всех наблюдателей о событии аудита.
//
// Каждый наблюдатель вызывается в отдельной горутине для предотвращения
// блокировки основного потока выполнения.
//
// Параметры:
//   - event: событие аудита для отправки наблюдателям
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

// FileAuditObserver реализует запись событий аудита в файл.
//
// Наблюдатель сериализует события в JSON формат и записывает
// их в указанный файл с автоматическим созданием директорий.
type FileAuditObserver struct {
	filePath string     // путь к файлу для записи
	mutex    sync.Mutex // мьютекс для безопасной записи в файл
}

// NewFileAuditObserver создает новый наблюдатель для записи в файл.
//
// Параметры:
//   - filePath: путь к файлу для записи событий аудита
//
// Возвращает настроенный FileAuditObserver.
func NewFileAuditObserver(filePath string) *FileAuditObserver {
	return &FileAuditObserver{
		filePath: filePath,
	}
}

// Notify записывает событие аудита в файл в формате JSON.
//
// Функция создает необходимые директории, открывает файл в режиме append
// и записывает событие как JSON строку с переносом строки.
//
// Параметры:
//   - event: событие аудита для записи
//
// Возвращает ошибку, если запись не удалась.
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

// HTTPAuditObserver реализует отправку событий аудита на удаленный сервер по HTTP.
//
// Наблюдатель сериализует события в JSON и отправляет их POST запросом
// на указанный URL с настроенным таймаутом.
type HTTPAuditObserver struct {
	url    string       // URL удаленного сервера
	client *http.Client // HTTP клиент с настроенным таймаутом
}

// NewHTTPAuditObserver создает новый наблюдатель для отправки по HTTP.
//
// Параметры:
//   - url: URL удаленного сервера для отправки событий аудита
//
// Возвращает настроенный HTTPAuditObserver с таймаутом 5 секунд.
func NewHTTPAuditObserver(url string) *HTTPAuditObserver {
	return &HTTPAuditObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify отправляет событие аудита на удаленный сервер по HTTP.
//
// Функция сериализует событие в JSON и отправляет POST запрос
// с заголовком Content-Type: application/json.
//
// Параметры:
//   - event: событие аудита для отправки
//
// Возвращает ошибку, если отправка не удалась или сервер вернул ошибку.
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
