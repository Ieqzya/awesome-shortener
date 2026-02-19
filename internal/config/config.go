// Package config предоставляет функциональность для управления конфигурацией сервиса сокращения URL.
//
// Пакет поддерживает конфигурацию через флаги командной строки и переменные окружения
// с правильным приоритетом: переменные окружения > флаги > значения по умолчанию.
package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Константы с дефолтными значениями
// DefaultServerAddress - адрес сервера по умолчанию
const DefaultServerAddress = "localhost:8080"

// DefaultBaseURL - базовый URL по умолчанию для сокращенных ссылок
const DefaultBaseURL = "http://localhost:8080"

// DefaultFileStoragePath - путь к файлу хранилища по умолчанию
const DefaultFileStoragePath = "/tmp/short-url-db.json"

// DefaultDatabaseDSN - строка подключения к базе данных по умолчанию (пустая)
const DefaultDatabaseDSN = ""

// Config содержит конфигурацию сервиса сокращения URL.
//
// Структура поддерживает настройку через флаги командной строки и переменные окружения.
// Приоритет параметров: переменные окружения > флаги > значения по умолчанию.
//
// Пример использования:
//
//	cfg, err := config.NewConfig()
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Server will start on %s\n", cfg.ServerAddress)
type Config struct {
	ServerAddress   string // адрес запуска HTTP-сервера (флаг -a, переменная SERVER_ADDRESS)
	BaseURL         string // базовый адрес результирующего сокращённого URL (флаг -b, переменная BASE_URL)
	FileStoragePath string // путь до файла с данными (флаг -f, переменная FILE_STORAGE_PATH)
	DatabaseDSN     string // строка подключения к базе данных (флаг -d, переменная DATABASE_DSN)
	AuditFile       string // путь к файлу аудита (флаг --audit-file, переменная AUDIT_FILE)
	AuditURL        string // URL удаленного сервера аудита (флаг --audit-url, переменная AUDIT_URL)
	EnableHTTPS     bool   // включить HTTPS (флаг -s, переменная ENABLE_HTTPS)
}

// NewConfig создает и инициализирует конфигурацию с приоритетом:
// 1. Переменные окружения (наивысший приоритет)
// 2. Флаги командной строки
// 3. Значения по умолчанию (наименьший приоритет)
//
// Поддерживаемые флаги:
//
//	-a: адрес запуска HTTP-сервера
//	-b: базовый адрес результирующего сокращённого URL
//	-f: путь до файла с данными
//	-d: строка подключения к базе данных
//	-s: включить HTTPS
//	--audit-file: путь к файлу аудита
//	--audit-url: URL удаленного сервера аудита
//
// Поддерживаемые переменные окружения:
//
//	SERVER_ADDRESS: адрес запуска HTTP-сервера
//	BASE_URL: базовый адрес результирующего сокращённого URL
//	FILE_STORAGE_PATH: путь до файла с данными
//	DATABASE_DSN: строка подключения к базе данных
//	ENABLE_HTTPS: включить HTTPS (true/false)
//	AUDIT_FILE: путь к файлу аудита
//	AUDIT_URL: URL удаленного сервера аудита
//
// Возвращает ошибку, если конфигурация не прошла валидацию.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	// Устанавливаем значения по умолчанию
	cfg.ServerAddress = DefaultServerAddress
	cfg.BaseURL = DefaultBaseURL
	cfg.FileStoragePath = DefaultFileStoragePath
	cfg.DatabaseDSN = DefaultDatabaseDSN

	// Парсим флаги командной строки
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "адрес запуска HTTP-сервера")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "базовый адрес результирующего сокращённого URL")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "путь до файла с данными")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "строка подключения к базе данных")
	flag.BoolVar(&cfg.EnableHTTPS, "s", false, "включить HTTPS")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "путь к файлу аудита")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "URL удаленного сервера аудита")
	flag.Parse()

	// Переопределяем переменными окружения (наивысший приоритет)
	if envServerAddr := strings.TrimSpace(os.Getenv("SERVER_ADDRESS")); envServerAddr != "" {
		cfg.ServerAddress = envServerAddr
	}

	if envBaseURL := strings.TrimSpace(os.Getenv("BASE_URL")); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}

	if envFileStoragePath := strings.TrimSpace(os.Getenv("FILE_STORAGE_PATH")); envFileStoragePath != "" {
		cfg.FileStoragePath = envFileStoragePath
	}

	if envDatabaseDSN := strings.TrimSpace(os.Getenv("DATABASE_DSN")); envDatabaseDSN != "" {
		cfg.DatabaseDSN = envDatabaseDSN
	}

	if envEnableHTTPS := strings.TrimSpace(os.Getenv("ENABLE_HTTPS")); envEnableHTTPS != "" {
		cfg.EnableHTTPS = envEnableHTTPS == "true" || envEnableHTTPS == "1"
	}

	if envAuditFile := strings.TrimSpace(os.Getenv("AUDIT_FILE")); envAuditFile != "" {
		cfg.AuditFile = envAuditFile
	}

	if envAuditURL := strings.TrimSpace(os.Getenv("AUDIT_URL")); envAuditURL != "" {
		cfg.AuditURL = envAuditURL
	}

	// Валидация конфигурации
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("ошибка валидации конфигурации: %w", err)
	}

	return cfg, nil
}

// validate проверяет корректность конфигурации
func (c *Config) validate() error {
	// Проверяем адрес сервера
	if strings.TrimSpace(c.ServerAddress) == "" {
		return fmt.Errorf("адрес сервера не может быть пустым")
	}

	// Проверяем базовый URL
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("базовый URL не может быть пустым")
	}

	// Проверяем, что базовый URL корректный
	parsedURL, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("некорректный базовый URL: %w", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("базовый URL должен содержать схему (http:// или https://)")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("базовый URL должен содержать хост")
	}

	// Убираем trailing slash если есть
	c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")

	return nil
}
