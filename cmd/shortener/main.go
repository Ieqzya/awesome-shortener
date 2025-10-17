package main

import (
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

func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 8)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func createShortURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(strings.TrimSpace(string(body))) == 0 {
		http.Error(w, "Bad Request", 400)
		return
	}

	originalURL := strings.TrimSpace(string(body))
	id := generateID()
	urls[id] = originalURL

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s/%s", cfg.BaseURL, id)
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
