package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"awesome-shortener/internal/config"
	"github.com/go-chi/chi/v5"
)

var urls = make(map[string]string)
var cfg *config.Config

// ShortenRequest представляет запрос на сокращение URL в JSON формате
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse представляет ответ с сокращенным URL в JSON формате
type ShortenResponse struct {
	Result string `json:"result"`
}

func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 8)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func createShortURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	
	var originalURL string
	
	// Обработка JSON запроса
	if strings.Contains(contentType, "application/json") {
		var req ShortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", 400)
			return
		}
		originalURL = strings.TrimSpace(req.URL)
	} else {
		// Обработка текстового запроса (как в предыдущих итерациях)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", 400)
			return
		}
		originalURL = strings.TrimSpace(string(body))
	}
	
	if originalURL == "" {
		http.Error(w, "Bad Request", 400)
		return
	}

	id := generateID()
	urls[id] = originalURL
	shortURL := fmt.Sprintf("%s/%s", cfg.BaseURL, id)

	// Возвращаем ответ в том же формате, что и запрос
	if strings.Contains(contentType, "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		response := ShortenResponse{Result: shortURL}
		json.NewEncoder(w).Encode(response)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(201)
		fmt.Fprint(w, shortURL)
	}
}

func redirectToOriginal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	if originalURL, ok := urls[id]; ok {
		w.Header().Set("Location", originalURL)
		w.WriteHeader(307)
	} else {
		http.Error(w, "Bad Request", 400)
	}
}

func main() {
	var err error
	cfg, err = config.NewConfig()
	if err != nil {
		log.Fatalf("Ошибка инициализации конфигурации: %v", err)
	}

	r := chi.NewRouter()
	
	r.Post("/", createShortURL)
	r.Get("/{id}", redirectToOriginal)

	fmt.Printf("Сервер запущен на %s\n", cfg.ServerAddress)
	fmt.Printf("Базовый URL: %s\n", cfg.BaseURL)
	
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
