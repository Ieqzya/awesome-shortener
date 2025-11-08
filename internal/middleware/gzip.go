package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipWriter обертка для ResponseWriter с поддержкой gzip сжатия
type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	// записываем данные в gzip writer
	return w.Writer.Write(b)
}

// compressReader обертка для чтения сжатых данных
type compressReader struct {
	io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		ReadCloser: r,
		zr:         zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.zr.Close(); err != nil {
		return err
	}
	return c.ReadCloser.Close()
}

// GzipMiddleware создает middleware для поддержки gzip сжатия
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, поддерживает ли клиент gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")

		// Обрабатываем входящие сжатые данные
		if r.Header.Get("Content-Encoding") == "gzip" {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		// Если клиент поддерживает gzip, сжимаем ответ
		if supportsGzip {
			// Создаем gzip writer
			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer gz.Close()

			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")

			// Создаем обертку для ResponseWriter
			gzw := gzipWriter{ResponseWriter: w, Writer: gz}

			// Перехватываем вызов для проверки Content-Type
			originalWriter := &contentTypeCheckWriter{
				ResponseWriter: gzw,
				gzipWriter:     gz,
				originalWriter: w,
			}

			next.ServeHTTP(originalWriter, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// contentTypeCheckWriter проверяет Content-Type и решает, нужно ли сжимать
type contentTypeCheckWriter struct {
	http.ResponseWriter
	gzipWriter     *gzip.Writer
	originalWriter http.ResponseWriter
	headerWritten  bool
}

func (w *contentTypeCheckWriter) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}
	w.headerWritten = true

	contentType := w.Header().Get("Content-Type")
	shouldCompress := strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/html") ||
		strings.Contains(contentType, "text/plain")

	if !shouldCompress {
		// Убираем заголовки сжатия и переключаемся на оригинальный writer
		w.originalWriter.Header().Del("Content-Encoding")
		w.originalWriter.Header().Del("Vary")
		w.ResponseWriter = w.originalWriter
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *contentTypeCheckWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}