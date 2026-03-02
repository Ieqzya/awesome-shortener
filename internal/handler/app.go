// Package handler содержит HTTP обработчики для сервиса сокращения URL.
//
// Пакет предоставляет структуру App с dependency injection и методы
// для обработки всех HTTP эндпоинтов: создание коротких URL,
// перенаправление, управление пользовательскими URL и проверка здоровья БД.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"awesome-shortener/internal/auth"
	"awesome-shortener/internal/config"
	"awesome-shortener/internal/model"
	"awesome-shortener/internal/service"
	"awesome-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
)

// App представляет основное приложение с внедренными зависимостями.
//
// Структура инкапсулирует все необходимые сервисы и конфигурацию
// для обработки HTTP запросов. Использует dependency injection
// для слабой связанности компонентов.
type App struct {
	config       *config.Config        // конфигурация приложения
	urlService   *service.URLService   // сервис для работы с URL
	authService  *auth.AuthService     // сервис аутентификации
	auditService *service.AuditService // сервис аудита
}

// NewApp создает новый экземпляр приложения с внедренными зависимостями.
//
// Функция инициализирует все сервисы и настраивает систему аудита
// в соответствии с конфигурацией.
//
// Параметры:
//   - cfg: конфигурация приложения
//   - store: реализация интерфейса Storage
//
// Возвращает настроенное приложение готовое к обработке запросов.
func NewApp(cfg *config.Config, store storage.Storage) *App {
	app := &App{
		config:       cfg,
		urlService:   service.NewURLService(store, cfg),
		authService:  auth.NewAuthService(),
		auditService: service.NewAuditService(),
	}

	// Настраиваем аудит
	app.setupAudit()

	return app
}

// setupAudit настраивает наблюдателей для аудита
func (app *App) setupAudit() {
	// Добавляем файловый аудит если указан путь
	if app.config.AuditFile != "" {
		fileObserver := service.NewFileAuditObserver(app.config.AuditFile)
		app.auditService.Subscribe(fileObserver)
	}

	// Добавляем HTTP аудит если указан URL
	if app.config.AuditURL != "" {
		httpObserver := service.NewHTTPAuditObserver(app.config.AuditURL)
		app.auditService.Subscribe(httpObserver)
	}
}

// Shutdown корректно останавливает приложение и освобождает ресурсы.
//
// Функция должна вызываться при завершении работы приложения
// для корректного завершения всех фоновых операций.
func (app *App) Shutdown() {
	if app.urlService != nil {
		app.urlService.Shutdown()
	}
}

// ShortenRequest представляет структуру запроса на сокращение URL в JSON формате.
//
// Используется для десериализации JSON запросов к эндпоинту /api/shorten.
type ShortenRequest struct {
	URL string `json:"url"` // URL для сокращения
}

// ShortenResponse представляет структуру ответа с сокращенным URL в JSON формате.
//
// Используется для сериализации JSON ответов от эндпоинта /api/shorten.
type ShortenResponse struct {
	Result string `json:"result"` // сокращенный URL
}

// BatchShortenRequest представляет элемент запроса для batch сокращения URL.
//
// Используется в массовых операциях сокращения нескольких URL за один запрос.
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"` // идентификатор для связи запроса и ответа
	OriginalURL   string `json:"original_url"`   // оригинальный URL для сокращения
}

// BatchShortenResponse представляет элемент ответа для batch сокращения URL.
//
// Используется в ответах на массовые операции сокращения URL.
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"` // идентификатор для связи запроса и ответа
	ShortURL      string `json:"short_url"`      // сокращенный URL
}

// auditShorten создает событие аудита для сокращения URL
func (app *App) auditShorten(userID, originalURL string) {
	if app.auditService == nil {
		return
	}

	event := model.AuditEvent{
		Timestamp: time.Now().Unix(),
		Action:    model.ActionShorten,
		UserID:    userID,
		URL:       originalURL,
	}

	app.auditService.NotifyAll(event)
}

// auditFollow создает событие аудита для перехода по ссылке
func (app *App) auditFollow(userID, originalURL string) {
	if app.auditService == nil {
		return
	}

	event := model.AuditEvent{
		Timestamp: time.Now().Unix(),
		Action:    model.ActionFollow,
		UserID:    userID,
		URL:       originalURL,
	}

	app.auditService.NotifyAll(event)
}

// shortenURL общая логика для сокращения URL
func (app *App) shortenURL(ctx context.Context, w http.ResponseWriter, r *http.Request, originalURL string) (string, int, error) {
	// Получаем или создаем ID пользователя
	userID := app.authService.GetOrCreateUserID(w, r)

	// Используем service для бизнес-логики
	shortURL, statusCode, err := app.urlService.ShortenURL(ctx, originalURL, userID)
	if err != nil && statusCode >= 500 {
		log.Printf("Ошибка сокращения URL: %v", err)
		return shortURL, statusCode, err
	}

	// Аудит успешного сокращения
	if statusCode < 400 {
		app.auditShorten(userID, originalURL)
	}

	return shortURL, statusCode, err
}

// CreateShortURL обрабатывает создание коротких URL для текстовых и JSON запросов.
//
// Эндпоинт поддерживает два формата:
//   - Текстовый: тело запроса содержит URL как plain text
//   - JSON: тело запроса содержит {"url": "http://example.com"}
//
// Автоматически определяет формат по заголовку Content-Type и возвращает
// ответ в том же формате. Создает или получает ID пользователя из cookie.
//
// HTTP методы: POST
// Пути: /
//
// Возможные ответы:
//   - 201 Created: URL успешно сокращен
//   - 409 Conflict: URL уже существует (возвращает существующий)
//   - 400 Bad Request: некорректный запрос
func (app *App) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	var originalURL string

	// Обработка JSON запроса
	if strings.Contains(contentType, "application/json") {
		var req ShortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		originalURL = strings.TrimSpace(req.URL)
	} else {
		// Обработка текстового запроса
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		originalURL = strings.TrimSpace(string(body))
	}

	shortURL, statusCode, err := app.shortenURL(r.Context(), w, r, originalURL)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Возвращаем ответ в том же формате, что и запрос
	if strings.Contains(contentType, "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		response := ShortenResponse{Result: shortURL}
		json.NewEncoder(w).Encode(response)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(statusCode)
		fmt.Fprint(w, shortURL)
	}
}

// CreateShortURLJSON обрабатывает создание коротких URL только для JSON запросов.
//
// Специализированный эндпоинт для JSON API, принимает только JSON запросы
// и возвращает только JSON ответы.
//
// HTTP методы: POST
// Пути: /api/shorten
//
// Формат запроса: {"url": "http://example.com"}
// Формат ответа: {"result": "http://localhost:8080/abc123"}
//
// Возможные ответы:
//   - 201 Created: URL успешно сокращен
//   - 409 Conflict: URL уже существует
//   - 400 Bad Request: некорректный JSON или пустой URL
//   - 500 Internal Server Error: ошибка сериализации ответа
func (app *App) CreateShortURLJSON(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest

	// Декодируем JSON из тела запроса
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	originalURL := strings.TrimSpace(req.URL)
	shortURL, statusCode, err := app.shortenURL(r.Context(), w, r, originalURL)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Возвращаем JSON ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ShortenResponse{Result: shortURL}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// CreateShortURLBatch обрабатывает массовое создание коротких URL.
//
// Эндпоинт принимает массив URL для одновременного сокращения,
// что повышает производительность при обработке множественных запросов.
//
// HTTP методы: POST
// Пути: /api/shorten/batch
//
// Формат запроса: [{"correlation_id": "1", "original_url": "http://example.com"}]
// Формат ответа: [{"correlation_id": "1", "short_url": "http://localhost:8080/abc123"}]
//
// Возможные ответы:
//   - 201 Created: все URL успешно сокращены
//   - 400 Bad Request: некорректный JSON или пустой массив
//   - 500 Internal Server Error: ошибка сохранения в хранилище
func (app *App) CreateShortURLBatch(w http.ResponseWriter, r *http.Request) {
	var requests []BatchShortenRequest

	// Декодируем JSON из тела запроса
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Проверяем, что батч не пустой
	if len(requests) == 0 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Получаем или создаем ID пользователя
	userID := app.authService.GetOrCreateUserID(w, r)

	// Подготавливаем данные для сохранения
	batchItems := make([]storage.BatchItem, 0, len(requests))
	responses := make([]BatchShortenResponse, 0, len(requests))

	for _, req := range requests {
		originalURL := strings.TrimSpace(req.OriginalURL)
		if originalURL == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Генерируем ID через service
		id, err := app.urlService.GenerateID()
		if err != nil {
			log.Printf("Ошибка генерации ID: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		shortURL := fmt.Sprintf("%s/%s", app.config.BaseURL, id)

		batchItems = append(batchItems, storage.BatchItem{
			CorrelationID: req.CorrelationID,
			ShortID:       id,
			OriginalURL:   originalURL,
		})

		responses = append(responses, BatchShortenResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      shortURL,
		})
	}

	// Сохраняем все URL в хранилище с user_id через urlService
	if err := app.urlService.SaveBatchWithUser(r.Context(), batchItems, userID); err != nil {
		log.Printf("Ошибка сохранения batch в хранилище: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Возвращаем JSON ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(responses); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// RedirectToOriginal обрабатывает перенаправление на оригинальный URL.
//
// Эндпоинт получает короткий идентификатор из URL пути,
// находит соответствующий оригинальный URL и выполняет перенаправление.
// Логирует событие аудита при успешном переходе.
//
// HTTP методы: GET
// Пути: /{id}
//
// Возможные ответы:
//   - 307 Temporary Redirect: успешное перенаправление
//   - 400 Bad Request: URL не найден
//   - 410 Gone: URL был удален
func (app *App) RedirectToOriginal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Используем service для получения URL
	originalURL, err := app.urlService.GetOriginalURL(r.Context(), id)
	if err != nil {
		// Проверяем, является ли ошибка удаленным URL
		if storage.IsDeletedError(err) {
			http.Error(w, "Gone", http.StatusGone)
			return
		}
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Аудит успешного перехода
	userID := app.authService.GetOrCreateUserID(w, r)
	app.auditFollow(userID, originalURL)

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// GetUserURLs возвращает все URL, принадлежащие текущему пользователю.
//
// Эндпоинт извлекает ID пользователя из cookie и возвращает
// список всех его сокращенных URL в JSON формате.
//
// HTTP методы: GET
// Пути: /api/user/urls
//
// Формат ответа: [{"short_url": "http://localhost:8080/abc123", "original_url": "http://example.com"}]
//
// Возможные ответы:
//   - 200 OK: список URL пользователя
//   - 204 No Content: у пользователя нет URL
//   - 500 Internal Server Error: ошибка получения данных
func (app *App) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	// Получаем или создаем ID пользователя
	userID := app.authService.GetOrCreateUserID(w, r)

	// Используем service для получения URL
	records, err := app.urlService.GetUserURLs(r.Context(), userID)
	if err != nil {
		log.Printf("Ошибка получения URL пользователя: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Если у пользователя нет URL
	if len(records) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Возвращаем JSON ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(records); err != nil {
		log.Printf("Ошибка кодирования JSON: %v", err)
	}
}

// DeleteUserURLs асинхронно удаляет указанные URL пользователя.
//
// Эндпоинт принимает массив коротких идентификаторов и помечает
// соответствующие URL как удаленные. Операция выполняется асинхронно
// для повышения производительности.
//
// HTTP методы: DELETE
// Пути: /api/user/urls
//
// Формат запроса: ["abc123", "def456"]
//
// Возможные ответы:
//   - 202 Accepted: запрос на удаление принят
//   - 400 Bad Request: некорректный JSON или пустой массив
func (app *App) DeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя
	userID := app.authService.GetOrCreateUserID(w, r)

	// Декодируем список ID для удаления
	var shortIDs []string
	if err := json.NewDecoder(r.Body).Decode(&shortIDs); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if len(shortIDs) == 0 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Используем service для асинхронного удаления с graceful shutdown
	app.urlService.DeleteURLsAsync(shortIDs, userID)

	// Возвращаем 202 Accepted
	w.WriteHeader(http.StatusAccepted)
}

// StatsResponse представляет структуру ответа для эндпоинта статистики
type StatsResponse struct {
	URLs  int `json:"urls"`  // количество сокращённых URL
	Users int `json:"users"` // количество пользователей
}

// GetStats возвращает статистику сервиса: количество URL и пользователей.
//
// Эндпоинт доступен только из доверенной подсети, указанной в конфигурации.
// IP-адрес клиента проверяется через заголовок X-Real-IP.
//
// HTTP методы: GET
// Пути: /api/internal/stats
//
// Формат ответа: {"urls": 100, "users": 50}
//
// Возможные ответы:
//   - 200 OK: статистика успешно получена
//   - 403 Forbidden: IP не в доверенной подсети
//   - 500 Internal Server Error: ошибка получения статистики
func (app *App) GetStats(w http.ResponseWriter, r *http.Request) {
	// Получаем статистику из хранилища
	urlsCount, usersCount, err := app.urlService.GetStats(r.Context())
	if err != nil {
		log.Printf("Ошибка получения статистики: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Возвращаем JSON ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := StatsResponse{
		URLs:  urlsCount,
		Users: usersCount,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Ошибка кодирования JSON: %v", err)
	}
}
