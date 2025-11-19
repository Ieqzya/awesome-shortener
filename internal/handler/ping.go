package handler

import (
	"context"
	"net/http"
	"time"

	"awesome-shortener/internal/storage"
)

// PingDatabase возвращает хендлер для проверки соединения с базой данных
func PingDatabase(db *storage.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Если база данных не настроена
		if db == nil {
			http.Error(w, "Database not configured", http.StatusInternalServerError)
			return
		}

		// Создаем контекст с таймаутом
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		// Проверяем соединение
		if err := db.Ping(ctx); err != nil {
			http.Error(w, "Database connection failed", http.StatusInternalServerError)
			return
		}

		// Успешная проверка
		w.WriteHeader(http.StatusOK)
	}
}
