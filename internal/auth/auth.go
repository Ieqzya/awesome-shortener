package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

const (
	cookieName       = "user_id"
	defaultSecretKey = "default-secret-key-please-change"
)

var secretKey string

func init() {
	// Получаем секретный ключ из переменной окружения
	secretKey = os.Getenv("SECRET_KEY")
	if secretKey == "" {
		secretKey = defaultSecretKey
		log.Println("WARNING: Using default secret key. Set SECRET_KEY environment variable for production!")
	}
}

// GenerateUserID генерирует новый уникальный ID пользователя
func GenerateUserID() string {
	return uuid.New().String()
}

// SignValue подписывает значение с помощью HMAC
func SignValue(value string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(value))
	signature := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s.%s", value, signature)
}

// VerifySignedValue проверяет подпись значения
func VerifySignedValue(signedValue string) (string, bool) {
	parts := strings.Split(signedValue, ".")
	if len(parts) != 2 {
		return "", false
	}

	value := parts[0]
	signature := parts[1]

	// Вычисляем ожидаемую подпись
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(value))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Сравниваем подписи
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return "", false
	}

	return value, true
}

// GetUserID извлекает ID пользователя из куки
func GetUserID(r *http.Request) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", err
	}

	userID, valid := VerifySignedValue(cookie.Value)
	if !valid {
		return "", fmt.Errorf("invalid cookie signature")
	}

	return userID, nil
}

// SetUserID устанавливает куку с ID пользователя
func SetUserID(w http.ResponseWriter, userID string) {
	signedValue := SignValue(userID)
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    signedValue,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400 * 30, // 30 дней
	}
	http.SetCookie(w, cookie)
}

// GetOrCreateUserID получает существующий или создает новый ID пользователя
func GetOrCreateUserID(w http.ResponseWriter, r *http.Request) string {
	userID, err := GetUserID(r)
	if err != nil || userID == "" {
		// Создаем новый ID
		userID = GenerateUserID()
		SetUserID(w, userID)
	}
	return userID
}
