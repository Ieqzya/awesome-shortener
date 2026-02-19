package config

import (
	"flag"
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	// Сохраняем оригинальные аргументы
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Сбрасываем флаги для чистого теста
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Тест с дефолтными значениями
	os.Args = []string{"cmd"}
	cfg, err := NewConfig()
	if err != nil {
		t.Errorf("Неожиданная ошибка: %v", err)
	}

	if cfg.ServerAddress != DefaultServerAddress {
		t.Errorf("Ожидали %s, получили %s", DefaultServerAddress, cfg.ServerAddress)
	}

	if cfg.BaseURL != DefaultBaseURL {
		t.Errorf("Ожидали %s, получили %s", DefaultBaseURL, cfg.BaseURL)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		serverAddress string
		baseURL       string
		expectError   bool
	}{
		{
			name:          "валидная конфигурация",
			serverAddress: "localhost:8080",
			baseURL:       "http://localhost:8080",
			expectError:   false,
		},
		{
			name:          "пустой адрес сервера",
			serverAddress: "",
			baseURL:       "http://localhost:8080",
			expectError:   true,
		},
		{
			name:          "пустой базовый URL",
			serverAddress: "localhost:8080",
			baseURL:       "",
			expectError:   true,
		},
		{
			name:          "некорректный базовый URL",
			serverAddress: "localhost:8080",
			baseURL:       "not-a-url",
			expectError:   true,
		},
		{
			name:          "базовый URL без схемы",
			serverAddress: "localhost:8080",
			baseURL:       "localhost:8080",
			expectError:   true,
		},
		{
			name:          "HTTPS URL",
			serverAddress: "localhost:8080",
			baseURL:       "https://example.com",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ServerAddress: tt.serverAddress,
				BaseURL:       tt.baseURL,
			}

			err := cfg.validate()
			if tt.expectError && err == nil {
				t.Errorf("Ожидали ошибку, но получили nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Не ожидали ошибку, но получили: %v", err)
			}
		})
	}
}

func TestBaseURLTrailingSlash(t *testing.T) {
	cfg := &Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080/",
	}

	err := cfg.validate()
	if err != nil {
		t.Errorf("Неожиданная ошибка: %v", err)
	}

	// Проверяем, что trailing slash убран
	if cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("Ожидали 'http://localhost:8080', получили '%s'", cfg.BaseURL)
	}
}

func TestEnableHTTPS(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		flagValue   bool
		expectHTTPS bool
	}{
		{
			name:        "HTTPS через флаг",
			envValue:    "",
			flagValue:   true,
			expectHTTPS: true,
		},
		{
			name:        "HTTPS через env (true)",
			envValue:    "true",
			flagValue:   false,
			expectHTTPS: true,
		},
		{
			name:        "HTTPS через env (1)",
			envValue:    "1",
			flagValue:   false,
			expectHTTPS: true,
		},
		{
			name:        "HTTPS отключен",
			envValue:    "false",
			flagValue:   false,
			expectHTTPS: false,
		},
		{
			name:        "env переопределяет флаг",
			envValue:    "true",
			flagValue:   false,
			expectHTTPS: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сохраняем оригинальные значения
			oldArgs := os.Args
			oldEnv := os.Getenv("ENABLE_HTTPS")
			defer func() {
				os.Args = oldArgs
				if oldEnv != "" {
					os.Setenv("ENABLE_HTTPS", oldEnv)
				} else {
					os.Unsetenv("ENABLE_HTTPS")
				}
			}()

			// Сбрасываем флаги
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Устанавливаем переменную окружения
			if tt.envValue != "" {
				os.Setenv("ENABLE_HTTPS", tt.envValue)
			} else {
				os.Unsetenv("ENABLE_HTTPS")
			}

			// Устанавливаем аргументы командной строки
			if tt.flagValue {
				os.Args = []string{"cmd", "-s"}
			} else {
				os.Args = []string{"cmd"}
			}

			cfg, err := NewConfig()
			if err != nil {
				t.Fatalf("Неожиданная ошибка: %v", err)
			}

			if cfg.EnableHTTPS != tt.expectHTTPS {
				t.Errorf("Ожидали EnableHTTPS=%v, получили %v", tt.expectHTTPS, cfg.EnableHTTPS)
			}
		})
	}
}
