package main

import (
	"fmt"
	"log"
	"net/http"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/handler"
	"awesome-shortener/internal/middleware"
	"awesome-shortener/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	// Инициализируем конфигурацию
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Ошибка инициализации конфигурации: %v", err)
	}

	// Выбираем хранилище с fallback логикой:
	// 1. PostgreSQL (если DATABASE_DSN задан)
	// 2. Файл (если FILE_STORAGE_PATH задан)
	// 3. Память (по умолчанию)
	var store storage.Storage
	var db *storage.Database

	// Пытаемся подключиться к PostgreSQL
	if cfg.DatabaseDSN != "" {
		db, err = storage.NewDatabase(cfg.DatabaseDSN)
		if err != nil {
			log.Printf("Предупреждение: не удалось подключиться к БД: %v", err)
		} else {
			store = db
			defer db.Close()
			fmt.Println("Хранилище: PostgreSQL")
		}
	}

	// Если БД не подключена, пытаемся использовать файл
	if store == nil && cfg.FileStoragePath != "" {
		fileStorage, err := storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			log.Printf("Предупреждение: не удалось открыть файл: %v", err)
		} else {
			store = fileStorage
			defer fileStorage.Close()
			fmt.Printf("Хранилище: файл (%s)\n", cfg.FileStoragePath)
		}
	}

	// Если ничего не подключено, используем память
	if store == nil {
		store = storage.NewMemoryStorage()
		fmt.Println("Хранилище: память")
	}

	// Создаем приложение с dependency injection
	app := handler.NewApp(cfg, store)

	// Инициализируем zap логгер
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Ошибка инициализации логгера: %v", err)
	}
	defer logger.Sync()

	r := chi.NewRouter()
	
	// Добавляем middleware для gzip сжатия
	r.Use(middleware.GzipMiddleware)
	// Добавляем middleware для логирования
	r.Use(middleware.Logger(logger))
	
	// Регистрируем обработчики
	r.Post("/", app.CreateShortURL)
	r.Post("/api/shorten", app.CreateShortURLJSON)
	r.Get("/{id}", app.RedirectToOriginal)
	
	// Добавляем хендлер для проверки соединения с БД
	r.Get("/ping", handler.PingDatabase(db))

	fmt.Printf("Сервер запущен на %s\n", cfg.ServerAddress)
	fmt.Printf("Базовый URL: %s\n", cfg.BaseURL)
	if cfg.DatabaseDSN != "" {
		fmt.Println("База данных: подключена")
	}
	
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
