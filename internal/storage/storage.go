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

// BatchItem представляет элемент для batch операции
type BatchItem struct {
	CorrelationID string
	ShortID       string
	OriginalURL   string
}

// Storage интерфейс для работы с хранилищем URL
type Storage interface {
	SaveURL(ctx context.Context, shortID, originalURL string) error
	GetURL(ctx context.Context, shortID string) (string, error)
	GetByOriginalURL(ctx context.Context, originalURL string) (string, error)
	SaveBatch(ctx context.Context, items []BatchItem) error
	Close() error
}
