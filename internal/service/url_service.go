package service

import (
	"context"
	cryptoRand "crypto/rand"
	"fmt"
	mathRand "math/rand"
	"sync"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/storage"
)

// URLService сервис для работы с URL
type URLService struct {
	storage storage.Storage
	config  *config.Config
	
	// Канал для graceful shutdown
	deleteChan chan deleteRequest
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

type deleteRequest struct {
	shortIDs []string
	userID   string
}

// NewURLService создает новый сервис
func NewURLService(store storage.Storage, cfg *config.Config) *URLService {
	ctx, cancel := context.WithCancel(context.Background())
	service := &URLService{
		storage:    store,
		config:     cfg,
		deleteChan: make(chan deleteRequest, 100),
		ctx:        ctx,
		cancel:     cancel,
	}
	
	// Запускаем воркер для обработки удалений
	service.wg.Add(1)
	go service.deleteWorker()
	
	return service
}

// GenerateID генерирует уникальный ID для URL
func (s *URLService) GenerateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 8)
	randomBytes := make([]byte, 8)
	
	// Используем crypto/rand для безопасной генерации
	if _, err := cryptoRand.Read(randomBytes); err != nil {
		// Fallback на менее безопасный вариант в случае ошибки
		for i := range result {
			result[i] = chars[mathRand.Intn(len(chars))]
		}
		return string(result)
	}
	
	for i := range result {
		result[i] = chars[int(randomBytes[i])%len(chars)]
	}
	return string(result)
}

// ShortenURL сокращает URL
func (s *URLService) ShortenURL(ctx context.Context, originalURL, userID string) (string, int, error) {
	if originalURL == "" {
		return "", 400, fmt.Errorf("URL не может быть пустым")
	}

	id := s.GenerateID()
	shortURL := fmt.Sprintf("%s/%s", s.config.BaseURL, id)
	
	// Сохраняем в хранилище с user_id
	err := s.storage.SaveURLWithUser(ctx, id, originalURL, userID)
	if err != nil {
		// Проверяем, является ли ошибка конфликтом
		var conflictErr *storage.ErrConflict
		if storage.IsConflictError(err) {
			// URL уже существует, возвращаем существующий short URL
			existingShortURL := fmt.Sprintf("%s/%s", s.config.BaseURL, conflictErr.ShortID)
			return existingShortURL, 409, nil
		}
		return "", 500, fmt.Errorf("ошибка сохранения")
	}

	return shortURL, 201, nil
}

// GetOriginalURL получает оригинальный URL
func (s *URLService) GetOriginalURL(ctx context.Context, shortID string) (string, error) {
	return s.storage.GetURL(ctx, shortID)
}

// GetUserURLs получает все URL пользователя
func (s *URLService) GetUserURLs(ctx context.Context, userID string) ([]storage.UserURLRecord, error) {
	records, err := s.storage.GetUserURLs(ctx, userID)
	if err != nil {
		return nil, err
	}
	
	// Преобразуем short_id в полные URL
	for i := range records {
		records[i].ShortURL = fmt.Sprintf("%s/%s", s.config.BaseURL, records[i].ShortURL)
	}
	
	return records, nil
}

// DeleteURLsAsync асинхронно удаляет URL
func (s *URLService) DeleteURLsAsync(shortIDs []string, userID string) {
	select {
	case s.deleteChan <- deleteRequest{shortIDs: shortIDs, userID: userID}:
	case <-s.ctx.Done():
		// Сервис завершается, игнорируем запрос
	}
}

// deleteWorker обрабатывает запросы на удаление
func (s *URLService) deleteWorker() {
	defer s.wg.Done()
	
	for {
		select {
		case req := <-s.deleteChan:
			if err := s.storage.DeleteURLs(context.Background(), req.shortIDs, req.userID); err != nil {
				// Логируем ошибку, но продолжаем работу
				fmt.Printf("Ошибка удаления URL: %v\n", err)
			}
		case <-s.ctx.Done():
			// Обрабатываем оставшиеся запросы перед завершением
			for {
				select {
				case req := <-s.deleteChan:
					if err := s.storage.DeleteURLs(context.Background(), req.shortIDs, req.userID); err != nil {
						fmt.Printf("Ошибка удаления URL: %v\n", err)
					}
				default:
					return
				}
			}
		}
	}
}

// Shutdown gracefully останавливает сервис
func (s *URLService) Shutdown() {
	s.cancel()
	s.wg.Wait()
	close(s.deleteChan)
}
