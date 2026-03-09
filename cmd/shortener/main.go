package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/grpc"
	pb "awesome-shortener/internal/grpc/pb"
	"awesome-shortener/internal/handler"
	"awesome-shortener/internal/middleware"
	"awesome-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	grpcServer "google.golang.org/grpc"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	// Выводим информацию о сборке
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
	fmt.Println()
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
	r.Post("/api/shorten/batch", app.CreateShortURLBatch)
	r.Get("/{id}", app.RedirectToOriginal)
	r.Get("/api/user/urls", app.GetUserURLs)
	r.Delete("/api/user/urls", app.DeleteUserURLs)

	// Добавляем хендлер для проверки соединения с БД
	r.Get("/ping", handler.PingDatabase(db))

	// Добавляем защищенный эндпоинт статистики
	r.With(middleware.TrustedSubnetMiddleware(cfg.TrustedSubnet)).Get("/api/internal/stats", app.GetStats)

	// Определяем протокол
	protocol := "HTTP"
	if cfg.EnableHTTPS {
		protocol = "HTTPS"
	}

	fmt.Printf("HTTP сервер запущен на %s (%s)\n", cfg.ServerAddress, protocol)
	fmt.Printf("Базовый URL: %s\n", cfg.BaseURL)
	if cfg.DatabaseDSN != "" {
		fmt.Println("База данных: подключена")
	}

	// Graceful shutdown
	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}

	// Запускаем HTTP сервер в отдельной горутине
	go func() {
		var err error
		if cfg.EnableHTTPS {
			// Запускаем HTTPS сервер
			err = server.ListenAndServeTLS("cert.pem", "key.pem")
		} else {
			// Запускаем HTTP сервер
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска HTTP сервера: %v", err)
		}
	}()

	// Запускаем gRPC сервер
	var grpcSrv *grpcServer.Server
	if cfg.GRPCAddress != "" {
		fmt.Printf("gRPC сервер запущен на %s\n", cfg.GRPCAddress)
		
		// Создаем gRPC сервер
		grpcSrv = grpcServer.NewServer()
		
		// Регистрируем сервис
		grpcService := grpc.NewShortenerServer(cfg, app.GetURLService(), app.GetAuthService())
		pb.RegisterShortenerServiceServer(grpcSrv, grpcService)
		
		// Запускаем gRPC сервер в отдельной горутине
		go func() {
			lis, err := net.Listen("tcp", cfg.GRPCAddress)
			if err != nil {
				log.Fatalf("Ошибка создания gRPC listener: %v", err)
			}
			
			if err := grpcSrv.Serve(lis); err != nil {
				log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
			}
		}()
	}

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-quit

	fmt.Printf("\nПолучен сигнал %v. Завершение работы серверов...\n", sig)

	// Останавливаем приложение (завершаем все фоновые операции)
	app.Shutdown()

	// Останавливаем gRPC сервер
	if grpcSrv != nil {
		fmt.Println("Остановка gRPC сервера...")
		grpcSrv.GracefulStop()
	}

	// Останавливаем HTTP сервер с таймаутом для обработки активных запросов
	fmt.Println("Остановка HTTP сервера...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Ошибка при завершении HTTP сервера: %v", err)
	}

	fmt.Println("Серверы остановлены")
}
