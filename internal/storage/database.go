package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
)

// Database представляет подключение к базе данных
type Database struct {
	db *sql.DB
}

// NewDatabase создает новое подключение к базе данных
func NewDatabase(dsn string) (*Database, error) {
	if dsn == "" {
		return nil, nil // База данных не настроена
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия соединения с БД: %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверка соединения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	database := &Database{db: db}

	// Выполняем миграции
	if err := database.runMigrations(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка выполнения миграций: %w", err)
	}

	return database, nil
}

// runMigrations выполняет миграции базы данных
func (d *Database) runMigrations(ctx context.Context) error {
	// Создаем таблицу urls
	query := `
		CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			short_id VARCHAR(255) UNIQUE NOT NULL,
			original_url TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_short_id ON urls(short_id);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_original_url ON urls(original_url);
	`

	_, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	return nil
}

// SaveURL сохраняет URL в базу данных
func (d *Database) SaveURL(ctx context.Context, shortID, originalURL string) error {
	query := `INSERT INTO urls (short_id, original_url) VALUES ($1, $2)`
	_, err := d.db.ExecContext(ctx, query, shortID, originalURL)
	if err != nil {
		// Проверяем, является ли ошибка нарушением уникальности
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pgerrcode.UniqueViolation {
			// Получаем существующий short_id для этого URL
			existingShortID, getErr := d.GetByOriginalURL(ctx, originalURL)
			if getErr != nil {
				return fmt.Errorf("ошибка получения существующего URL: %w", getErr)
			}
			return &ErrConflict{ShortID: existingShortID}
		}
		return fmt.Errorf("ошибка сохранения URL: %w", err)
	}
	return nil
}

// GetURL получает оригинальный URL по короткому ID
func (d *Database) GetURL(ctx context.Context, shortID string) (string, error) {
	var originalURL string
	query := `SELECT original_url FROM urls WHERE short_id = $1`
	err := d.db.QueryRowContext(ctx, query, shortID).Scan(&originalURL)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("URL не найден")
	}
	if err != nil {
		return "", fmt.Errorf("ошибка получения URL: %w", err)
	}
	return originalURL, nil
}

// GetByOriginalURL получает короткий ID по оригинальному URL
func (d *Database) GetByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	var shortID string
	query := `SELECT short_id FROM urls WHERE original_url = $1`
	err := d.db.QueryRowContext(ctx, query, originalURL).Scan(&shortID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("URL не найден")
	}
	if err != nil {
		return "", fmt.Errorf("ошибка получения short_id: %w", err)
	}
	return shortID, nil
}

// SaveBatch сохраняет множество URL в одной транзакции
func (d *Database) SaveBatch(ctx context.Context, items []BatchItem) error {
	if len(items) == 0 {
		return nil
	}

	// Начинаем транзакцию
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// Подготавливаем запрос
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO urls (short_id, original_url) VALUES ($1, $2)`)
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса: %w", err)
	}
	defer stmt.Close()

	// Выполняем вставку для каждого элемента
	for _, item := range items {
		if _, err := stmt.ExecContext(ctx, item.ShortID, item.OriginalURL); err != nil {
			return fmt.Errorf("ошибка сохранения URL: %w", err)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}

// Ping проверяет соединение с базой данных
func (d *Database) Ping(ctx context.Context) error {
	if d == nil || d.db == nil {
		return fmt.Errorf("база данных не инициализирована")
	}
	return d.db.PingContext(ctx)
}

// Close закрывает соединение с базой данных
func (d *Database) Close() error {
	if d == nil || d.db == nil {
		return nil
	}
	return d.db.Close()
}

// DB возвращает объект *sql.DB для выполнения запросов
func (d *Database) DB() *sql.DB {
	if d == nil {
		return nil
	}
	return d.db
}
