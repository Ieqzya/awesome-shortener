package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
)

// Простое хранилище URL в памяти
var urlStorage = make(map[string]string)

// Функция для генерации случайного ID
func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 8)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// Обработчик для создания короткой ссылки (POST /)
func createShortURL(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод
	if r.Method != http.MethodPost {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	originalURL := strings.TrimSpace(string(body))
	
	// Проверяем, что URL не пустой
	if originalURL == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Генерируем ID и сохраняем
	id := generateID()
	urlStorage[id] = originalURL

	// Возвращаем короткую ссылку
	shortURL := fmt.Sprintf("http://localhost:8080/%s", id)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// Обработчик для перенаправления (GET /{id})
func redirectToOriginal(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод
	if r.Method != http.MethodGet {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Получаем ID из пути
	path := r.URL.Path
	if path == "/" {
		// Если путь просто "/", то это POST запрос для создания ссылки
		createShortURL(w, r)
		return
	}

	id := strings.TrimPrefix(path, "/")
	
	// Проверяем, что ID не пустой
	if id == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Ищем оригинальный URL
	originalURL, exists := urlStorage[id]
	if !exists {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Перенаправляем
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {
	// Настраиваем маршруты
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/" {
			createShortURL(w, r)
		} else if r.Method == http.MethodGet && r.URL.Path != "/" {
			redirectToOriginal(w, r)
		} else {
			http.Error(w, "Bad Request", http.StatusBadRequest)
		}
	})

	// Запускаем сервер
	fmt.Println("Сервер запущен на http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("Ошибка запуска сервера: %v\n", err)
	}
}
