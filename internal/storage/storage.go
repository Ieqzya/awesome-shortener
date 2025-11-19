package storage

import "context"

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
	SaveBatch(ctx context.Context, items []BatchItem) error
	Close() error
}
