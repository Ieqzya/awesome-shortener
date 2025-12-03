package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"awesome-shortener/internal/config"
	"awesome-shortener/internal/storage"
)

func TestGetUserURLs_NoAuth(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)

	// Запрос без куки
	req := httptest.NewRequest("GET", "/api/user/urls", nil)
	w := httptest.NewRecorder()

	app.GetUserURLs(w, req)

	if w.Code != 401 {
		t.Errorf("Ожидали код 401, получили %d", w.Code)
	}
}

func TestGetUserURLs_NoURLs(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)

	// Сначала создаем URL, чтобы получить куку
	body := strings.NewReader("https://example.com")
	req1 := httptest.NewRequest("POST", "/", body)
	w1 := httptest.NewRecorder()
	app.CreateShortURL(w1, req1)

	// Получаем куку из ответа
	cookies := w1.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Кука не была установлена")
	}

	// Очищаем хранилище
	store = storage.NewMemoryStorage()
	app = NewApp(cfg, store)

	// Запрос с кукой, но без URL
	req2 := httptest.NewRequest("GET", "/api/user/urls", nil)
	req2.AddCookie(cookies[0])
	w2 := httptest.NewRecorder()

	app.GetUserURLs(w2, req2)

	if w2.Code != 204 {
		t.Errorf("Ожидали код 204, получили %d", w2.Code)
	}
}

func TestGetUserURLs_Success(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)

	// Создаем несколько URL
	urls := []string{
		"https://example1.com",
		"https://example2.com",
		"https://example3.com",
	}

	var cookie *httptest.ResponseRecorder
	for _, url := range urls {
		body := strings.NewReader(url)
		req := httptest.NewRequest("POST", "/", body)
		
		// Если есть кука, добавляем её
		if cookie != nil {
			cookies := cookie.Result().Cookies()
			if len(cookies) > 0 {
				req.AddCookie(cookies[0])
			}
		}
		
		w := httptest.NewRecorder()
		app.CreateShortURL(w, req)
		cookie = w
	}

	// Получаем куку
	cookies := cookie.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Кука не была установлена")
	}

	// Запрашиваем URL пользователя
	req := httptest.NewRequest("GET", "/api/user/urls", nil)
	req.AddCookie(cookies[0])
	w := httptest.NewRecorder()

	app.GetUserURLs(w, req)

	if w.Code != 200 {
		t.Errorf("Ожидали код 200, получили %d", w.Code)
	}

	// Проверяем формат ответа
	var records []storage.URLRecord
	if err := json.NewDecoder(w.Body).Decode(&records); err != nil {
		t.Fatalf("Ошибка декодирования JSON: %v", err)
	}

	if len(records) != 3 {
		t.Errorf("Ожидали 3 URL, получили %d", len(records))
	}

	// Проверяем формат URL
	for _, record := range records {
		if !strings.HasPrefix(record.ShortURL, "http://localhost:8080/") {
			t.Errorf("Неправильный формат short_url: %s", record.ShortURL)
		}
		if record.OriginalURL == "" {
			t.Error("original_url не должен быть пустым")
		}
	}
}

func TestUserIsolation(t *testing.T) {
	cfg := &config.Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}
	store := storage.NewMemoryStorage()
	app := NewApp(cfg, store)

	// Пользователь 1 создает URL
	body1 := strings.NewReader("https://user1.com")
	req1 := httptest.NewRequest("POST", "/", body1)
	w1 := httptest.NewRecorder()
	app.CreateShortURL(w1, req1)
	cookie1 := w1.Result().Cookies()[0]

	// Пользователь 2 создает URL
	body2 := strings.NewReader("https://user2.com")
	req2 := httptest.NewRequest("POST", "/", body2)
	w2 := httptest.NewRecorder()
	app.CreateShortURL(w2, req2)
	cookie2 := w2.Result().Cookies()[0]

	// Пользователь 1 запрашивает свои URL
	reqGet1 := httptest.NewRequest("GET", "/api/user/urls", nil)
	reqGet1.AddCookie(cookie1)
	wGet1 := httptest.NewRecorder()
	app.GetUserURLs(wGet1, reqGet1)

	var records1 []storage.URLRecord
	json.NewDecoder(wGet1.Body).Decode(&records1)

	// Пользователь 2 запрашивает свои URL
	reqGet2 := httptest.NewRequest("GET", "/api/user/urls", nil)
	reqGet2.AddCookie(cookie2)
	wGet2 := httptest.NewRecorder()
	app.GetUserURLs(wGet2, reqGet2)

	var records2 []storage.URLRecord
	json.NewDecoder(wGet2.Body).Decode(&records2)

	// Проверяем изоляцию
	if len(records1) != 1 || len(records2) != 1 {
		t.Errorf("Каждый пользователь должен видеть только свой URL")
	}

	if records1[0].OriginalURL != "https://user1.com" {
		t.Errorf("Пользователь 1 видит неправильный URL: %s", records1[0].OriginalURL)
	}

	if records2[0].OriginalURL != "https://user2.com" {
		t.Errorf("Пользователь 2 видит неправильный URL: %s", records2[0].OriginalURL)
	}
}
