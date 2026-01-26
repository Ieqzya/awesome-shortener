// Package service содержит бизнес-логику сервиса сокращения URL.
//
// Пакет предоставляет сервисы для работы с URL (URLService) и аудита (AuditService).
// URLService инкапсулирует логику сокращения URL, генерации идентификаторов
// и управления пользовательскими URL. AuditService реализует паттерн Observer
// для логирования действий пользователей.
package service

import (
	"context"
	cryptoRand "crypto/rand"
	"errors"
	"fmt"
	mathRand "math/rand"
	"sync"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/storage"
)

// URLService предоставляет бизнес-логику для работы с сокращенными URL.
//
// Сервис инкапсулирует операции создания, получения и удаления URL,
// а также управляет асинхронным удалением с graceful shutdown.
// Использует криптографически стойкую генерацию идентификаторов.
type URLService struct {
	storage storage.Storage // хранилище URL
	config  *config.Config  // конфигурация сервиса

	// Канал для graceful shutdown
	deleteChan chan deleteRequest // канал для запросов на удаление
	wg         sync.WaitGroup      // группа ожидания для graceful shutdown
	ctx        context.Context     // контекст для отмены операций
	cancel     context.CancelFunc  // функция отмены контекста
}

type deleteRequest struct {
	shortIDs []string
	userID   string
}

// NewURLService создает новый экземпляр URLService.
//
// Функция инициализирует сервис с переданным хранилищем и конфигурацией,
// запускает фоновый воркер для обработки асинхронных удалений.
//
// Параметры:
//   - store: реализация интерфейса Storage для хранения URL
//   - cfg: конфигурация сервиса
//
// Возвращает настроенный URLService с запущенным воркером.
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

// GenerateID генерирует криптографически стойкий уникальный идентификатор для URL.
//
// Функция использует crypto/rand для генерации случайных байтов,
// которые затем преобразуются в строку из алфавитно-цифровых символов.
// В случае ошибки crypto/rand используется fallback на math/rand.
//
// Возвращает строку длиной 8 символов, содержащую буквы и цифры.
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

// ShortenURL создает сокращенный URL для переданного оригинального URL.
//
// Функция генерирует уникальный идентификатор, создает полный короткий URL
// и сохраняет его в хранилище с привязкой к пользователю.
// Обрабатывает конфликты при попытке сохранить дублирующийся URL.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - originalURL: оригинальный URL для сокращения
//   - userID: идентификатор пользователя
//
// Возвращает:
//   - string: полный сокращенный URL
//   - int: HTTP код статуса (201 для нового URL, 409 для существующего)
//   - error: ошибка выполнения операции
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
		if errors.As(err, &conflictErr) {
			// URL уже существует, возвращаем существующий short URL
			existingShortURL := fmt.Sprintf("%s/%s", s.config.BaseURL, conflictErr.ShortID)
			return existingShortURL, 409, nil
		}
		return "", 500, fmt.Errorf("ошибка сохранения")
	}

	return shortURL, 201, nil
}

// GetOriginalURL получает оригинальный URL по короткому идентификатору.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: короткий идентификатор URL
//
// Возвращает:
//   - string: оригинальный URL
//   - error: ошибка, если URL не найден или удален
func (s *URLService) GetOriginalURL(ctx context.Context, shortID string) (string, error) {
	return s.storage.GetURL(ctx, shortID)
}

// GetUserURLs получает все URL, принадлежащие указанному пользователю.
//
// Функция извлекает записи из хранилища и преобразует короткие идентификаторы
// в полные URL с использованием базового адреса из конфигурации.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - userID: идентификатор пользователя
//
// Возвращает:
//   - []storage.UserURLRecord: список URL пользователя
//   - error: ошибка выполнения операции
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

// DeleteURLsAsync асинхронно удаляет указанные URL пользователя.
//
// Функция отправляет запрос на удаление в канал для обработки
// фоновым воркером. Не блокирует выполнение.
//
// Параметры:
//   - shortIDs: список коротких идентификаторов для удаления
//   - userID: идентификатор пользователя
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

// Shutdown корректно останавливает сервис и завершает все фоновые операции.
//
// Функция отменяет контекст, ожидает завершения всех горутин
// и закрывает канал для запросов на удаление.
// Должна вызываться при завершении работы приложения.
func (s *URLService) Shutdown() {
	s.cancel()
	s.wg.Wait()
	close(s.deleteChan)
}
