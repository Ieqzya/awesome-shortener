// Package storage предоставляет интерфейсы и реализации для хранения URL.
//
// Пакет содержит интерфейс Storage и его реализации для различных
// типов хранилищ: память, файл и PostgreSQL база данных.
// Также определяет специальные типы ошибок для обработки конфликтов
// и удаленных URL.
package storage

import (
	"context"
	"errors"
)

// ErrConflict представляет ошибку при попытке сохранить дублирующийся URL.
//
// Ошибка возникает, когда пользователь пытается сократить URL,
// который уже существует в системе. Содержит ShortID существующего URL.
type ErrConflict struct {
	ShortID string // идентификатор существующего сокращенного URL
}

func (e *ErrConflict) Error() string {
	return "URL already exists"
}

// IsConflictError проверяет, является ли ошибка конфликтом дублирующихся URL.
//
// Параметры:
//   - err: ошибка для проверки
//
// Возвращает true, если ошибка является ErrConflict.
func IsConflictError(err error) bool {
	var conflictErr *ErrConflict
	return errors.As(err, &conflictErr)
}

// ErrDeleted представляет ошибку при попытке получить удаленный URL.
//
// Ошибка возникает, когда пользователь пытается перейти по URL,
// который был помечен как удаленный.
type ErrDeleted struct{}

func (e *ErrDeleted) Error() string {
	return "URL has been deleted"
}

// IsDeletedError проверяет, является ли ошибка попыткой доступа к удаленному URL.
//
// Параметры:
//   - err: ошибка для проверки
//
// Возвращает true, если ошибка является ErrDeleted.
func IsDeletedError(err error) bool {
	var deletedErr *ErrDeleted
	return errors.As(err, &deletedErr)
}

// BatchItem представляет элемент для batch операции сохранения URL.
//
// Используется при массовом сохранении нескольких URL за одну операцию.
type BatchItem struct {
	CorrelationID string // идентификатор корреляции для связи запроса и ответа
	ShortID       string // короткий идентификатор URL
	OriginalURL   string // оригинальный URL
}

// UserURLRecord представляет запись URL пользователя для API ответов.
//
// Используется при возврате списка URL пользователя через API.
type UserURLRecord struct {
	ShortURL    string `json:"short_url"`   // полный сокращенный URL
	OriginalURL string `json:"original_url"` // оригинальный URL
}

// Storage определяет интерфейс для работы с хранилищем URL.
//
// Интерфейс поддерживает различные операции: сохранение, получение,
// удаление URL, а также работу с пользователями и batch операции.
// Реализации должны быть потокобезопасными.
type Storage interface {
	// SaveURL сохраняет URL без привязки к пользователю (для обратной совместимости).
	SaveURL(ctx context.Context, shortID, originalURL string) error

	// SaveURLWithUser сохраняет URL с привязкой к пользователю.
	SaveURLWithUser(ctx context.Context, shortID, originalURL, userID string) error

	// GetURL получает оригинальный URL по короткому идентификатору.
	// Возвращает ErrDeleted, если URL был удален.
	GetURL(ctx context.Context, shortID string) (string, error)

	// GetByOriginalURL получает короткий идентификатор по оригинальному URL.
	GetByOriginalURL(ctx context.Context, originalURL string) (string, error)

	// GetUserURLs получает все URL указанного пользователя.
	GetUserURLs(ctx context.Context, userID string) ([]UserURLRecord, error)

	// SaveBatch сохраняет множество URL без привязки к пользователю.
	SaveBatch(ctx context.Context, items []BatchItem) error

	// SaveBatchWithUser сохраняет множество URL с привязкой к пользователю.
	SaveBatchWithUser(ctx context.Context, items []BatchItem, userID string) error

	// DeleteURLs помечает URL как удаленные (мягкое удаление).
	// Удаляет только URL, принадлежащие указанному пользователю.
	DeleteURLs(ctx context.Context, shortIDs []string, userID string) error

	// IsDeleted проверяет, помечен ли URL как удаленный.
	IsDeleted(ctx context.Context, shortID string) (bool, error)

	// Close закрывает соединение с хранилищем и освобождает ресурсы.
	Close() error
}
