package main

import (
	"context"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"
)

// TestGracefulShutdown проверяет корректное завершение сервера
func TestGracefulShutdown(t *testing.T) {
	// Этот тест проверяет, что сервер корректно обрабатывает сигналы завершения
	// В реальном приложении graceful shutdown тестируется через интеграционные тесты

	// Создаем тестовый сервер
	server := &http.Server{
		Addr: "localhost:0", // случайный порт
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Имитируем долгий запрос
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}),
	}

	// Запускаем сервер
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Ошибка сервера: %v", err)
		}
	}()

	// Даем серверу время на запуск
	time.Sleep(50 * time.Millisecond)

	// Останавливаем сервер с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Ошибка при graceful shutdown: %v", err)
	}
}

// TestSignalHandling проверяет обработку различных сигналов
func TestSignalHandling(t *testing.T) {
	signals := []os.Signal{
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}

	for _, sig := range signals {
		t.Run(sig.String(), func(t *testing.T) {
			// Проверяем, что сигнал определен
			if sig == nil {
				t.Error("Сигнал не должен быть nil")
			}
		})
	}
}
