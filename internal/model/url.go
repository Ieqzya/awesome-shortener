package model

// URLRecord представляет запись о сокращенном URL в системе.
//
// Структура используется для сериализации и десериализации
// данных о URL в JSON формате при работе с файловым хранилищем.
//
// Пример использования:
//
//	record := model.URLRecord{
//		UUID:        "1",
//		ShortURL:    "abc123",
//		OriginalURL: "https://example.com",
//	}
type URLRecord struct {
	UUID        string `json:"uuid"`         // уникальный идентификатор записи
	ShortURL    string `json:"short_url"`    // короткий идентификатор URL
	OriginalURL string `json:"original_url"` // оригинальный URL
}
