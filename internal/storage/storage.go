package storage

import (
	"context"
	"errors"
)

// ErrConflict ошибка при попытке сохранить дублирующийся URL
type ErrConflict struct {
	ShortID string
}

func (e *ErrConflict) Error() string {
	return "URL already exists"
}

// IsConflictError проверяет, является ли ошибка конфликтом
func IsConflictError(err error) bool {
	var conflictErr *ErrConflict
	return errors.As(err, &conflictErr)
}

// ErrDeleted ошибка при попытке получить удаленный URL
type ErrDeleted struct{}

func (e *ErrDeleted) Error() string {
	return "URL has been deleted"
}

// IsDeletedError проверяет, является ли ошибка удаленным URL
func IsDeletedError(err error) bool {
	var deletedErr *ErrDeleted
	return errors.As(err, &deletedErr)
}

// BatchItem представляет элемент для batch операции
type BatchItem struct {
	CorrelationID string
	ShortID       string
	OriginalURL   string
}

// UserURLRecord представляет запись URL пользователя
type UserURLRecord struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// Storage интерфейс для работы с хранилищем URL
type Storage interface {
	SaveURL(ctx context.Context, shortID, originalURL string) error
	SaveURLWithUser(ctx context.Context, shortID, originalURL, userID string) error
	GetURL(ctx context.Context, shortID string) (string, error)
	GetByOriginalURL(ctx context.Context, originalURL string) (string, error)
	GetUserURLs(ctx context.Context, userID string) ([]UserURLRecord, error)
	SaveBatch(ctx context.Context, items []BatchItem) error
	SaveBatchWithUser(ctx context.Context, items []BatchItem, userID string) error
	DeleteURLs(ctx context.Context, shortIDs []string, userID string) error
	IsDeleted(ctx context.Context, shortID string) (bool, error)
	Close() error
}
