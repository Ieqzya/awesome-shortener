package storage

import "context"

// Storage интерфейс для работы с хранилищем URL
type Storage interface {
	SaveURL(ctx context.Context, shortID, originalURL string) error
	GetURL(ctx context.Context, shortID string) (string, error)
	Close() error
}
