package handler

import (
	"context"
	"net/http"
	"time"

	"awesome-shortener/internal/storage"
)

// PingDatabase возвращает HTTP обработчик для проверки соединения с базой данных.
//
// Эндпоинт предназначен для health check'ов и мониторинга состояния
// подключения к базе данных. Выполняет ping с таймаутом 1 секунда.
//
// HTTP методы: GET
// Пути: /ping
//
// Параметры:
//   - db: экземпляр Database для проверки (может быть nil)
//
// Возможные ответы:
//   - 200 OK: соединение с БД работает
//   - 500 Internal Server Error: БД не настроена или соединение не работает
//
// Пример использования:
//
//	r.Get("/ping", handler.PingDatabase(db))
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
