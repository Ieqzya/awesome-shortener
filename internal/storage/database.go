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
			user_id VARCHAR(255),
			is_deleted BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_short_id ON urls(short_id);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_original_url ON urls(original_url);
		CREATE INDEX IF NOT EXISTS idx_user_id ON urls(user_id);
		CREATE INDEX IF NOT EXISTS idx_is_deleted ON urls(is_deleted);
	`

	_, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы: %w", err)
	}

	return nil
}

// SaveURL сохраняет URL в базу данных без user_id
func (d *Database) SaveURL(ctx context.Context, shortID, originalURL string) error {
	return d.SaveURLWithUser(ctx, shortID, originalURL, "")
}

// SaveURLWithUser сохраняет URL в базу данных с user_id
func (d *Database) SaveURLWithUser(ctx context.Context, shortID, originalURL, userID string) error {
	query := `INSERT INTO urls (short_id, original_url, user_id) VALUES ($1, $2, $3)`
	_, err := d.db.ExecContext(ctx, query, shortID, originalURL, userID)
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
	var isDeleted bool
	query := `SELECT original_url, is_deleted FROM urls WHERE short_id = $1`
	err := d.db.QueryRowContext(ctx, query, shortID).Scan(&originalURL, &isDeleted)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("URL не найден")
	}
	if err != nil {
		return "", fmt.Errorf("ошибка получения URL: %w", err)
	}
	if isDeleted {
		return "", &ErrDeleted{}
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

// GetUserURLs получает все URL пользователя
func (d *Database) GetUserURLs(ctx context.Context, userID string) ([]UserURLRecord, error) {
	query := `SELECT short_id, original_url FROM urls WHERE user_id = $1`
	rows, err := d.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения URL пользователя: %w", err)
	}
	defer rows.Close()

	var records []UserURLRecord
	for rows.Next() {
		var shortID, originalURL string
		if err := rows.Scan(&shortID, &originalURL); err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		records = append(records, UserURLRecord{
			ShortURL:    shortID, // Будет преобразовано в полный URL в хендлере
			OriginalURL: originalURL,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по строкам: %w", err)
	}

	return records, nil
}

// SaveBatch сохраняет множество URL в одной транзакции без user_id
func (d *Database) SaveBatch(ctx context.Context, items []BatchItem) error {
	return d.SaveBatchWithUser(ctx, items, "")
}

// SaveBatchWithUser сохраняет множество URL в одной транзакции с user_id
func (d *Database) SaveBatchWithUser(ctx context.Context, items []BatchItem, userID string) error {
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
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO urls (short_id, original_url, user_id) VALUES ($1, $2, $3)`)
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса: %w", err)
	}
	defer stmt.Close()

	// Выполняем вставку для каждого элемента
	for _, item := range items {
		if _, err := stmt.ExecContext(ctx, item.ShortID, item.OriginalURL, userID); err != nil {
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

// DeleteURLs помечает URL как удаленные (batch update)
func (d *Database) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	if len(shortIDs) == 0 {
		return nil
	}

	// Используем batch update для эффективности
	query := `UPDATE urls SET is_deleted = TRUE WHERE short_id = ANY($1) AND user_id = $2`
	_, err := d.db.ExecContext(ctx, query, pq.Array(shortIDs), userID)
	if err != nil {
		return fmt.Errorf("ошибка удаления URL: %w", err)
	}

	return nil
}

// IsDeleted проверяет, удален ли URL
func (d *Database) IsDeleted(ctx context.Context, shortID string) (bool, error) {
	var isDeleted bool
	query := `SELECT is_deleted FROM urls WHERE short_id = $1`
	err := d.db.QueryRowContext(ctx, query, shortID).Scan(&isDeleted)
	if err == sql.ErrNoRows {
		return false, fmt.Errorf("URL не найден")
	}
	if err != nil {
		return false, fmt.Errorf("ошибка проверки статуса URL: %w", err)
	}
	return isDeleted, nil
}
