package auth

import (
	"net/http/httptest"
	"testing"
)

func TestGenerateUserID(t *testing.T) {
	authService := NewAuthService()

	id1 := authService.GenerateUserID()
	id2 := authService.GenerateUserID()

	if id1 == "" {
		t.Error("ID не должен быть пустым")
	}

	if id1 == id2 {
		t.Error("ID должны быть уникальными")
	}
}

func TestSignAndVerifyValue(t *testing.T) {
	authService := NewAuthService()
	value := "test-user-id"

	// Подписываем значение
	signed := authService.SignValue(value)

	// Проверяем подпись
	verified, valid := authService.VerifySignedValue(signed)

	if !valid {
		t.Error("Подпись должна быть валидной")
	}

	if verified != value {
		t.Errorf("Ожидали '%s', получили '%s'", value, verified)
	}
}

func TestVerifySignedValue_Invalid(t *testing.T) {
	authService := NewAuthService()

	// Невалидная подпись
	_, valid := authService.VerifySignedValue("invalid.signature")
	if valid {
		t.Error("Невалидная подпись не должна проходить проверку")
	}

	// Подделанная подпись
	_, valid = authService.VerifySignedValue("test-id.fakesignature")
	if valid {
		t.Error("Подделанная подпись не должна проходить проверку")
	}

	// Неправильный формат
	_, valid = authService.VerifySignedValue("no-dot-separator")
	if valid {
		t.Error("Неправильный формат не должен проходить проверку")
	}
}

func TestSetAndGetUserID(t *testing.T) {
	authService := NewAuthService()
	userID := "test-user-123"

	// Создаем тестовый запрос и ответ
	w := httptest.NewRecorder()

	// Устанавливаем куку
	authService.SetUserID(w, userID)

	// Получаем куку из ответа
	result := w.Result()
	defer result.Body.Close()
	cookies := result.Cookies()

	if len(cookies) == 0 {
		t.Fatal("Кука не была установлена")
	}

	cookie := cookies[0]

	// Проверяем параметры куки
	if cookie.Name != cookieName {
		t.Errorf("Неправильное имя куки: %s", cookie.Name)
	}

	if !cookie.HttpOnly {
		t.Error("Кука должна быть HttpOnly")
	}

	if cookie.Path != "/" {
		t.Errorf("Неправильный путь куки: %s", cookie.Path)
	}

	// Создаем новый запрос с этой кукой
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	// Получаем ID из куки
	retrievedID, err := authService.GetUserID(req)
	if err != nil {
		t.Fatalf("Ошибка получения ID: %v", err)
	}

	if retrievedID != userID {
		t.Errorf("Ожидали '%s', получили '%s'", userID, retrievedID)
	}
}

func TestGetUserID_NoCookie(t *testing.T) {
	authService := NewAuthService()
	req := httptest.NewRequest("GET", "/", nil)

	_, err := authService.GetUserID(req)
	if err == nil {
		t.Error("Должна быть ошибка при отсутствии куки")
	}
}

func TestGetOrCreateUserID_NewUser(t *testing.T) {
	authService := NewAuthService()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Получаем или создаем ID
	userID := authService.GetOrCreateUserID(w, req)

	if userID == "" {
		t.Error("ID не должен быть пустым")
	}

	// Проверяем, что кука была установлена
	result := w.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	if len(cookies) == 0 {
		t.Error("Кука должна быть установлена для нового пользователя")
	}
}

func TestGetOrCreateUserID_ExistingUser(t *testing.T) {
	authService := NewAuthService()

	// Создаем пользователя
	userID := "existing-user-123"
	w1 := httptest.NewRecorder()
	authService.SetUserID(w1, userID)
	result := w1.Result()
	defer result.Body.Close()
	cookie := result.Cookies()[0]

	// Создаем запрос с существующей кукой
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)
	w2 := httptest.NewRecorder()

	// Получаем ID
	retrievedID := authService.GetOrCreateUserID(w2, req)

	if retrievedID != userID {
		t.Errorf("Ожидали '%s', получили '%s'", userID, retrievedID)
	}
}
