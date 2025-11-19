package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
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

	return &Database{db: db}, nil
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
