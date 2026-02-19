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

func TestJSONConfig(t *testing.T) {
	// Создаем временный JSON файл
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Не удалось создать временный файл: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Записываем тестовую конфигурацию
	jsonData := `{
		"server_address": "localhost:9090",
		"base_url": "http://example.com",
		"file_storage_path": "/custom/path.json",
		"database_dsn": "postgres://localhost/test",
		"enable_https": true
	}`
	if _, err := tmpFile.WriteString(jsonData); err != nil {
		t.Fatalf("Не удалось записать в файл: %v", err)
	}
	tmpFile.Close()

	// Сохраняем оригинальные значения
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Устанавливаем путь к конфигу через флаг
	os.Args = []string{"cmd", "-c", tmpFile.Name()}

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("Неожиданная ошибка: %v", err)
	}

	// Проверяем значения из JSON
	if cfg.ServerAddress != "localhost:9090" {
		t.Errorf("Ожидали ServerAddress='localhost:9090', получили '%s'", cfg.ServerAddress)
	}
	if cfg.BaseURL != "http://example.com" {
		t.Errorf("Ожидали BaseURL='http://example.com', получили '%s'", cfg.BaseURL)
	}
	if cfg.FileStoragePath != "/custom/path.json" {
		t.Errorf("Ожидали FileStoragePath='/custom/path.json', получили '%s'", cfg.FileStoragePath)
	}
	if cfg.DatabaseDSN != "postgres://localhost/test" {
		t.Errorf("Ожидали DatabaseDSN='postgres://localhost/test', получили '%s'", cfg.DatabaseDSN)
	}
	if !cfg.EnableHTTPS {
		t.Error("Ожидали EnableHTTPS=true, получили false")
	}
}

func TestConfigPriority(t *testing.T) {
	// Создаем временный JSON файл
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Не удалось создать временный файл: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// JSON конфигурация
	jsonData := `{
		"server_address": "localhost:9090",
		"base_url": "http://json.com"
	}`
	if _, err := tmpFile.WriteString(jsonData); err != nil {
		t.Fatalf("Не удалось записать в файл: %v", err)
	}
	tmpFile.Close()

	// Сохраняем оригинальные значения
	oldArgs := os.Args
	oldEnv := os.Getenv("SERVER_ADDRESS")
	defer func() {
		os.Args = oldArgs
		if oldEnv != "" {
			os.Setenv("SERVER_ADDRESS", oldEnv)
		} else {
			os.Unsetenv("SERVER_ADDRESS")
		}
	}()

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Устанавливаем переменную окружения (наивысший приоритет)
	os.Setenv("SERVER_ADDRESS", "localhost:7777")

	// Устанавливаем флаг и путь к конфигу
	os.Args = []string{"cmd", "-c", tmpFile.Name(), "-a", "localhost:8888"}

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("Неожиданная ошибка: %v", err)
	}

	// Переменная окружения должна иметь наивысший приоритет
	if cfg.ServerAddress != "localhost:7777" {
		t.Errorf("Ожидали ServerAddress='localhost:7777' (из env), получили '%s'", cfg.ServerAddress)
	}

	// BaseURL должен быть из JSON (нет флага и env)
	if cfg.BaseURL != "http://json.com" {
		t.Errorf("Ожидали BaseURL='http://json.com' (из JSON), получили '%s'", cfg.BaseURL)
	}
}

func TestJSONConfigWithEnv(t *testing.T) {
	// Создаем временный JSON файл
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Не удалось создать временный файл: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	jsonData := `{"server_address": "localhost:9090"}`
	if _, err := tmpFile.WriteString(jsonData); err != nil {
		t.Fatalf("Не удалось записать в файл: %v", err)
	}
	tmpFile.Close()

	// Сохраняем оригинальные значения
	oldArgs := os.Args
	oldConfigEnv := os.Getenv("CONFIG")
	defer func() {
		os.Args = oldArgs
		if oldConfigEnv != "" {
			os.Setenv("CONFIG", oldConfigEnv)
		} else {
			os.Unsetenv("CONFIG")
		}
	}()

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Устанавливаем путь к конфигу через переменную окружения
	os.Setenv("CONFIG", tmpFile.Name())
	os.Args = []string{"cmd"}

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("Неожиданная ошибка: %v", err)
	}

	if cfg.ServerAddress != "localhost:9090" {
		t.Errorf("Ожидали ServerAddress='localhost:9090', получили '%s'", cfg.ServerAddress)
	}
}

func TestInvalidJSONConfig(t *testing.T) {
	// Создаем временный файл с невалидным JSON
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Не удалось создать временный файл: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Невалидный JSON
	if _, err := tmpFile.WriteString("{invalid json}"); err != nil {
		t.Fatalf("Не удалось записать в файл: %v", err)
	}
	tmpFile.Close()

	// Сохраняем оригинальные значения
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	os.Args = []string{"cmd", "-c", tmpFile.Name()}

	_, err = NewConfig()
	if err == nil {
		t.Error("Ожидали ошибку при невалидном JSON, но получили nil")
	}
}

func TestNonExistentJSONConfig(t *testing.T) {
	// Сохраняем оригинальные значения
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Сбрасываем флаги
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	os.Args = []string{"cmd", "-c", "/nonexistent/config.json"}

	_, err := NewConfig()
	if err == nil {
		t.Error("Ожидали ошибку при несуществующем файле, но получили nil")
	}
}
