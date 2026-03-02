// Package config предоставляет функциональность для управления конфигурацией сервиса сокращения URL.
//
// Пакет поддерживает конфигурацию через JSON файл, флаги командной строки и переменные окружения
// с правильным приоритетом: переменные окружения > флаги > JSON файл > значения по умолчанию.
package config

import (
	"encoding/json"
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
// Структура поддерживает настройку через JSON файл, флаги командной строки и переменные окружения.
// Приоритет параметров: переменные окружения > флаги > JSON файл > значения по умолчанию.
//
// Пример использования:
//
//	cfg, err := config.NewConfig()
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Server will start on %s\n", cfg.ServerAddress)
type Config struct {
	ServerAddress   string `json:"server_address"`    // адрес запуска HTTP-сервера
	BaseURL         string `json:"base_url"`          // базовый адрес результирующего сокращённого URL
	FileStoragePath string `json:"file_storage_path"` // путь до файла с данными
	DatabaseDSN     string `json:"database_dsn"`      // строка подключения к базе данных
	AuditFile       string `json:"audit_file"`        // путь к файлу аудита
	AuditURL        string `json:"audit_url"`         // URL удаленного сервера аудита
	EnableHTTPS     bool   `json:"enable_https"`      // включить HTTPS
	TrustedSubnet   string `json:"trusted_subnet"`    // доверенная подсеть в формате CIDR
}

// JSONConfig представляет структуру JSON файла конфигурации с опциональными полями
type JSONConfig struct {
	ServerAddress   *string `json:"server_address,omitempty"`
	BaseURL         *string `json:"base_url,omitempty"`
	FileStoragePath *string `json:"file_storage_path,omitempty"`
	DatabaseDSN     *string `json:"database_dsn,omitempty"`
	AuditFile       *string `json:"audit_file,omitempty"`
	AuditURL        *string `json:"audit_url,omitempty"`
	EnableHTTPS     *bool   `json:"enable_https,omitempty"`
	TrustedSubnet   *string `json:"trusted_subnet,omitempty"`
}

// NewConfig создает и инициализирует конфигурацию с приоритетом:
// 1. Переменные окружения (наивысший приоритет)
// 2. Флаги командной строки
// 3. JSON файл конфигурации
// 4. Значения по умолчанию (наименьший приоритет)
//
// Поддерживаемые флаги:
//
//	-a: адрес запуска HTTP-сервера
//	-b: базовый адрес результирующего сокращённого URL
//	-f: путь до файла с данными
//	-d: строка подключения к базе данных
//	-s: включить HTTPS
//	-t: доверенная подсеть в формате CIDR
//	-c/-config: путь к JSON файлу конфигурации
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
//	TRUSTED_SUBNET: доверенная подсеть в формате CIDR
//	CONFIG: путь к JSON файлу конфигурации
//	AUDIT_FILE: путь к файлу аудита
//	AUDIT_URL: URL удаленного сервера аудита
//
// Возвращает ошибку, если конфигурация не прошла валидацию.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	// 1. Устанавливаем значения по умолчанию
	cfg.ServerAddress = DefaultServerAddress
	cfg.BaseURL = DefaultBaseURL
	cfg.FileStoragePath = DefaultFileStoragePath
	cfg.DatabaseDSN = DefaultDatabaseDSN

	// 2. Парсим флаги командной строки
	var configFile string
	flag.StringVar(&configFile, "c", "", "путь к JSON файлу конфигурации")
	flag.StringVar(&configFile, "config", "", "путь к JSON файлу конфигурации")
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "адрес запуска HTTP-сервера")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "базовый адрес результирующего сокращённого URL")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "путь до файла с данными")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "строка подключения к базе данных")
	flag.BoolVar(&cfg.EnableHTTPS, "s", false, "включить HTTPS")
	flag.StringVar(&cfg.TrustedSubnet, "t", "", "доверенная подсеть в формате CIDR")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "путь к файлу аудита")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "URL удаленного сервера аудита")
	flag.Parse()

	// Проверяем переменную окружения для пути к конфигу
	if envConfigFile := strings.TrimSpace(os.Getenv("CONFIG")); envConfigFile != "" {
		configFile = envConfigFile
	}

	// 3. Загружаем конфигурацию из JSON файла (если указан)
	if configFile != "" {
		if err := loadJSONConfig(configFile, cfg); err != nil {
			return nil, fmt.Errorf("ошибка загрузки конфигурации из файла: %w", err)
		}
	}

	// 4. Переопределяем переменными окружения (наивысший приоритет)
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

	if envTrustedSubnet := strings.TrimSpace(os.Getenv("TRUSTED_SUBNET")); envTrustedSubnet != "" {
		cfg.TrustedSubnet = envTrustedSubnet
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

// loadJSONConfig загружает конфигурацию из JSON файла
func loadJSONConfig(filename string, cfg *Config) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл: %w", err)
	}

	var jsonCfg JSONConfig
	if err := json.Unmarshal(data, &jsonCfg); err != nil {
		return fmt.Errorf("не удалось распарсить JSON: %w", err)
	}

	// Применяем значения из JSON только если они указаны
	if jsonCfg.ServerAddress != nil {
		cfg.ServerAddress = *jsonCfg.ServerAddress
	}
	if jsonCfg.BaseURL != nil {
		cfg.BaseURL = *jsonCfg.BaseURL
	}
	if jsonCfg.FileStoragePath != nil {
		cfg.FileStoragePath = *jsonCfg.FileStoragePath
	}
	if jsonCfg.DatabaseDSN != nil {
		cfg.DatabaseDSN = *jsonCfg.DatabaseDSN
	}
	if jsonCfg.AuditFile != nil {
		cfg.AuditFile = *jsonCfg.AuditFile
	}
	if jsonCfg.AuditURL != nil {
		cfg.AuditURL = *jsonCfg.AuditURL
	}
	if jsonCfg.EnableHTTPS != nil {
		cfg.EnableHTTPS = *jsonCfg.EnableHTTPS
	}
	if jsonCfg.TrustedSubnet != nil {
		cfg.TrustedSubnet = *jsonCfg.TrustedSubnet
	}

	return nil
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
