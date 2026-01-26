package model

// AuditEvent представляет событие аудита
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
