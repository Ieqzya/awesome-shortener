package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateShortURL(t *testing.T) {
	urlStorage = make(map[string]string)

	body := strings.NewReader("https://google.com")
	req := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()

	createShortURL(w, req)

	if w.Code != 201 {
		t.Errorf("Ожидали код 201, получили %d", w.Code)
	}

	response := w.Body.String()
	if !strings.Contains(response, "http://localhost:8080/") {
		t.Errorf("Неправильный ответ: %s", response)
	}

	if len(urlStorage) != 1 {
		t.Errorf("URL не сохранился")
	}
}

func TestCreateShortURLBadRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(""))
	w := httptest.NewRecorder()

	createShortURL(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400, получили %d", w.Code)
	}
}

func TestRedirectToOriginal(t *testing.T) {
	urlStorage = make(map[string]string)
	urlStorage["test123"] = "https://example.com"

	req := httptest.NewRequest("GET", "/test123", nil)
	w := httptest.NewRecorder()

	redirectToOriginal(w, req)

	if w.Code != 307 {
		t.Errorf("Ожидали код 307, получили %d", w.Code)
	}

	if w.Header().Get("Location") != "https://example.com" {
		t.Errorf("Неправильный Location")
	}
}

func TestRedirectNotFound(t *testing.T) {
	urlStorage = make(map[string]string)

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()

	redirectToOriginal(w, req)

	if w.Code != 400 {
		t.Errorf("Ожидали код 400, получили %d", w.Code)
	}
}