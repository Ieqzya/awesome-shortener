package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Константы с дефолтными значениями
const (
	DefaultServerAddress = "localhost:8080"
	DefaultBaseURL       = "http://localhost:8080"
)

// Config содержит конфигурацию сервиса
type Config struct {
	ServerAddress string // адрес запуска HTTP-сервера
	BaseURL       string // базовый адрес результирующего сокращённого URL
}

// NewConfig создает и инициализирует конфигурацию с приоритетом:
// 1. Переменные окружения
// 2. Флаги командной строки
// 3. Значения по умолчанию
func NewConfig() (*Config, error) {
	cfg := &Config{}

	// Устанавливаем значения по умолчанию
	cfg.ServerAddress = DefaultServerAddress
	cfg.BaseURL = DefaultBaseURL

	// Парсим флаги командной строки
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "адрес запуска HTTP-сервера")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "базовый адрес результирующего сокращённого URL")
	flag.Parse()

	// Переопределяем переменными окружения (наивысший приоритет)
	if envServerAddr := os.Getenv("SERVER_ADDRESS"); envServerAddr != "" {
		cfg.ServerAddress = envServerAddr
	}

	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
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