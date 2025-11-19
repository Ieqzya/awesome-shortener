package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/storage"
	"github.com/go-chi/chi/v5"
)

// App представляет приложение с зависимостями
type App struct {
	config  *config.Config
	storage storage.Storage
}

// NewApp создает новое приложение
func NewApp(cfg *config.Config, store storage.Storage) *App {
	return &App{
		config:  cfg,
		storage: store,
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

func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 8)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// shortenURL общая логика для сокращения URL
func (app *App) shortenURL(ctx context.Context, originalURL string) (string, error) {
	if originalURL == "" {
		return "", fmt.Errorf("URL не может быть пустым")
	}

	id := generateID()
	shortURL := fmt.Sprintf("%s/%s", app.config.BaseURL, id)
	
	// Сохраняем в хранилище
	if err := app.storage.SaveURL(ctx, id, originalURL); err != nil {
		log.Printf("Ошибка сохранения в хранилище: %v", err)
		return "", fmt.Errorf("ошибка сохранения")
	}

	return shortURL, nil
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
	
	shortURL, err := app.shortenURL(r.Context(), originalURL)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Возвращаем ответ в том же формате, что и запрос
	if strings.Contains(contentType, "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		response := ShortenResponse{Result: shortURL}
		json.NewEncoder(w).Encode(response)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
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
	shortURL, err := app.shortenURL(r.Context(), originalURL)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Возвращаем JSON ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	
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
		
		// Генерируем ID
		id := generateID()
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
	
	// Сохраняем все URL в хранилище
	if err := app.storage.SaveBatch(r.Context(), batchItems); err != nil {
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
	
	originalURL, err := app.storage.GetURL(r.Context(), id)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}