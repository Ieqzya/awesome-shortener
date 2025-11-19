package main

import (
	"fmt"
	"log"
	"net/http"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/handler"
	"awesome-shortener/internal/middleware"
	"awesome-shortener/internal/service"
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

	// Инициализируем подключение к базе данных (если настроено)
	db, err := storage.NewDatabase(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	if db != nil {
		defer db.Close()
		fmt.Println("Подключение к базе данных установлено")
	}

	// Инициализируем файловое хранилище
	fileStorage := service.NewFileStorage(cfg.FileStoragePath)
	defer fileStorage.Close() // Закрываем файл при завершении

	// Создаем приложение с dependency injection
	app := handler.NewApp(cfg, fileStorage)

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
