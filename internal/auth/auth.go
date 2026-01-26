// Package auth предоставляет функциональность для аутентификации пользователей
// в сервисе сокращения URL.
//
// Пакет реализует систему аутентификации на основе подписанных cookie,
// используя HMAC-SHA256 для обеспечения целостности данных.
// Каждый пользователь получает уникальный UUID, который сохраняется
// в подписанной cookie для последующей идентификации.
//
// Пример использования:
//
//	// Получение или создание ID пользователя
//	userID := auth.GetOrCreateUserID(w, r)
//
//	// Проверка существующего ID
//	if existingID, err := auth.GetUserID(r); err == nil {
//		fmt.Printf("Пользователь: %s\n", existingID)
//	}
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

// GenerateUserID генерирует новый уникальный идентификатор пользователя.
//
// Функция использует UUID v4 для создания криптографически стойкого
// уникального идентификатора.
//
// Возвращает строковое представление UUID.
func GenerateUserID() string {
	return uuid.New().String()
}

// SignValue подписывает значение с помощью HMAC-SHA256.
//
// Функция создает подпись для переданного значения, используя
// секретный ключ из переменной окружения SECRET_KEY.
// Подпись добавляется к значению через точку.
//
// Параметры:
//   - value: значение для подписи
//
// Возвращает подписанное значение в формате "value.signature".
func SignValue(value string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(value))
	signature := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s.%s", value, signature)
}

// VerifySignedValue проверяет подпись значения.
//
// Функция извлекает значение и подпись из подписанной строки,
// вычисляет ожидаемую подпись и сравнивает её с переданной.
//
// Параметры:
//   - signedValue: подписанное значение в формате "value.signature"
//
// Возвращает:
//   - string: оригинальное значение (если подпись верна)
//   - bool: true, если подпись корректна
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

// GetUserID извлекает идентификатор пользователя из cookie.
//
// Функция читает cookie с именем "user_id", проверяет её подпись
// и возвращает идентификатор пользователя.
//
// Параметры:
//   - r: HTTP запрос
//
// Возвращает:
//   - string: идентификатор пользователя
//   - error: ошибка, если cookie отсутствует или подпись неверна
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

// SetUserID устанавливает cookie с идентификатором пользователя.
//
// Функция создает подписанную cookie с переданным идентификатором
// пользователя. Cookie устанавливается с флагом HttpOnly для безопасности
// и сроком действия 30 дней.
//
// Параметры:
//   - w: HTTP ответ
//   - userID: идентификатор пользователя для сохранения
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

// GetOrCreateUserID получает существующий или создает новый идентификатор пользователя.
//
// Функция сначала пытается получить идентификатор из существующей cookie.
// Если cookie отсутствует или повреждена, создается новый идентификатор
// и устанавливается соответствующая cookie.
//
// Параметры:
//   - w: HTTP ответ (для установки cookie при создании нового ID)
//   - r: HTTP запрос (для чтения существующей cookie)
//
// Возвращает идентификатор пользователя (существующий или новый).
func GetOrCreateUserID(w http.ResponseWriter, r *http.Request) string {
	userID, err := GetUserID(r)
	if err != nil || userID == "" {
		// Создаем новый ID
		userID = GenerateUserID()
		SetUserID(w, userID)
	}
	return userID
}
