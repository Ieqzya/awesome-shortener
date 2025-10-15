package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var urls = make(map[string]string)

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
	fmt.Fprintf(w, "http://localhost:8080/%s", id)
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
	rand.Seed(time.Now().UnixNano())

	r := chi.NewRouter()
	
	r.Post("/", createShortURL)
	r.Get("/{id}", redirectToOriginal)

	fmt.Println("Сервер на http://localhost:8080")
	http.ListenAndServe(":8080", r)
}
