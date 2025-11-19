package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// responseWriter обертка для захвата информации об ответе
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Logger создает middleware для логирования HTTP запросов и ответов
func Logger(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Создаем обертку для ResponseWriter
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // по умолчанию 200
			}

			// Выполняем следующий обработчик
			next.ServeHTTP(wrapped, r)

			// Вычисляем время выполнения
			duration := time.Since(start)

			// Логируем информацию о запросе и ответе
			logger.Info("HTTP request",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Duration("duration", duration),
				zap.Int("status", wrapped.statusCode),
				zap.Int("size", wrapped.size),
			)
		})
	}
}