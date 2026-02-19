package repository

// Storage интерфейс для хранения URL
type Storage interface {
	Store(shortURL, originalURL string) error
	Get(shortURL string) (string, bool)
}
