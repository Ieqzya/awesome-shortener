// Package model содержит модели данных для сервиса сокращения URL.
//
// Пакет предоставляет структуры для представления URL записей,
// событий аудита и других сущностей системы.
package model

// AuditEvent представляет событие аудита в системе сокращения URL.
//
// Структура используется для логирования действий пользователей,
// таких как создание коротких URL и переходы по ссылкам.
//
// Пример использования:
//
//	event := model.AuditEvent{
//		Timestamp: time.Now().Unix(),
//		Action:    model.ActionShorten,
//		UserID:    "user123",
//		URL:       "https://example.com",
//	}
type AuditEvent struct {
	Timestamp int64  `json:"ts"`      // unix timestamp события
	Action    string `json:"action"`  // действие: shorten или follow
	UserID    string `json:"user_id"` // идентификатор пользователя
	URL       string `json:"url"`     // оригинальный URL
}

// Константы для типов действий
const (
	ActionShorten = "shorten" // создание короткого URL
	ActionFollow  = "follow"  // переход по короткому URL
)
