package main

import (
	"fmt"
	"log"
	"net/http"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/handler"
	"awesome-shortener/internal/middleware"
	"awesome-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	// Инициализируем конфигурацию
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Ошибка инициализации конфигурации: %v", err)
	}

	// Инициализируем файловое хранилище
	storage := service.NewFileStorage(cfg.FileStoragePath)
	defer storage.Close() // Закрываем файл при завершении

	// Создаем приложение с dependency injection
	app := handler.NewApp(cfg, storage)

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

	fmt.Printf("Сервер запущен на %s\n", cfg.ServerAddress)
	fmt.Printf("Базовый URL: %s\n", cfg.BaseURL)
	
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
