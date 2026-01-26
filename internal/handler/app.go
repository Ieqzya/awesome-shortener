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

// App представляет приложение с зависимостями
type App struct {
	config       *config.Config
	urlService   *service.URLService
	storage      storage.Storage // Оставляем для обратной совместимости с тестами
	auditService *service.AuditService
}

// NewApp создает новое приложение
func NewApp(cfg *config.Config, store storage.Storage) *App {
	app := &App{
		config:       cfg,
		urlService:   service.NewURLService(store, cfg),
		storage:      store,
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

// Shutdown gracefully останавливает приложение
func (app *App) Shutdown() {
	if app.urlService != nil {
		app.urlService.Shutdown()
	}
}

// ShortenRequest представляет запрос на сокращение URL в JSON формате
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse представляет ответ с сокращенным URL в JSON формате
type ShortenResponse struct {
	Result string `json:"result"`
}

// BatchShortenRequest представляет элемент запроса для batch сокращения
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchShortenResponse представляет элемент ответа для batch сокращения
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
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
	userID := auth.GetOrCreateUserID(w, r)

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

// CreateShortURL обрабатывает создание короткого URL (текстовый и JSON)
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

// CreateShortURLJSON обрабатывает создание короткого URL только для JSON
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

// CreateShortURLBatch обрабатывает batch создание коротких URL
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
		id := app.urlService.GenerateID()
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

	// Получаем или создаем ID пользователя
	userID := auth.GetOrCreateUserID(w, r)

	// Сохраняем все URL в хранилище с user_id
	if err := app.storage.SaveBatchWithUser(r.Context(), batchItems, userID); err != nil {
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

// RedirectToOriginal обрабатывает перенаправление на оригинальный URL
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
	userID := auth.GetOrCreateUserID(w, r)
	app.auditFollow(userID, originalURL)

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// GetUserURLs возвращает все URL пользователя
func (app *App) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	// Получаем или создаем ID пользователя
	userID := auth.GetOrCreateUserID(w, r)

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

// DeleteUserURLs асинхронно удаляет URL пользователя
func (app *App) DeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя
	userID := auth.GetOrCreateUserID(w, r)

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
