# cmd/shortener

В данной директории содержится код, который скомпилируется в бинарное приложение.

Рекомендуется помещать только код, необходимый для запуска приложения, но не бизнес-логику.

Название директории должно соответствовать названию приложения.

Директория `cmd/shortener` содержит:
- точку входа в приложение (функция `main`)
- инициализацию зависимостей (можно вынести в отдельный пакет `internal/app`)
- настройку и запуск HTTP-сервера (можно вынести в отдельный пакет `internal/router`)
- обработку сигналов завершения работы приложения

## Переменные сборки

Приложение поддерживает встраивание информации о версии через ldflags:

```go
var (
    buildVersion = "N/A"  // Версия сборки
    buildDate    = "N/A"  // Дата сборки
    buildCommit  = "N/A"  // Хеш коммита
)
```

### Сборка с версией

```bash
# Простая сборка (значения по умолчанию "N/A")
go build -o shortener main.go

# Сборка с версией
go build -ldflags "\
  -X main.buildVersion=v1.0.0 \
  -X main.buildDate=2026-02-14 \
  -X main.buildCommit=abc123d" \
  -o shortener main.go

# Автоматическая сборка с git информацией
go build -ldflags "\
  -X main.buildVersion=$(git describe --tags --always) \
  -X main.buildDate=$(date -u +%Y-%m-%d) \
  -X main.buildCommit=$(git rev-parse --short HEAD)" \
  -o shortener main.go
```

### Использование скрипта сборки

В корне проекта есть скрипт `build.sh`:

```bash
# Сборка с автоматическим определением версии
./build.sh

# Сборка с кастомной версией
VERSION=v2.0.0 ./build.sh
```

### Вывод при запуске

При старте приложение выводит информацию о сборке:

```
Build version: v1.0.0
Build date: 2026-02-14
Build commit: abc123d

Хранилище: файл (/tmp/short-url-db.json)
Сервер запущен на localhost:8080 (HTTP)
Базовый URL: http://localhost:8080
```

Если переменные не были установлены при сборке, выводится "N/A":

```
Build version: N/A
Build date: N/A
Build commit: N/A
```

## HTTPS поддержка

Приложение поддерживает запуск с HTTPS.

### Генерация сертификата

Для работы HTTPS необходимы файлы `cert.pem` и `key.pem` в директории `cmd/shortener/`.

Используйте скрипт для генерации самоподписанного сертификата:

```bash
# Из корня проекта
./generate-cert.sh
```

Или вручную:

```bash
openssl req -x509 -newkey rsa:2048 \
  -keyout cmd/shortener/key.pem \
  -out cmd/shortener/cert.pem \
  -days 365 \
  -nodes \
  -subj "/CN=localhost"
```

### Запуск с HTTPS

```bash
# Используя флаг -s
./shortener -s

# Используя переменную окружения
ENABLE_HTTPS=true ./shortener

# Комбинация с другими параметрами
./shortener -s -a localhost:8443 -b https://localhost:8443
```

### Проверка HTTPS

```bash
# С самоподписанным сертификатом (игнорируем проверку)
curl -k https://localhost:8080/

# Создание короткой ссылки
curl -k -X POST https://localhost:8080/ \
  -H "Content-Type: text/plain" \
  -d "https://example.com"
```

При запуске с HTTPS в выводе будет указан протокол:

```
Сервер запущен на localhost:8080 (HTTPS)
```

## Конфигурация через JSON

Приложение поддерживает конфигурацию через JSON файл.

### Формат файла

Создайте файл `config.json`:

```json
{
    "server_address": "localhost:8080",
    "base_url": "http://localhost:8080",
    "file_storage_path": "/tmp/short-url-db.json",
    "database_dsn": "postgres://user:pass@localhost/shortener",
    "enable_https": false
}
```

Все поля опциональны. Если поле не указано, используется значение по умолчанию.

### Использование

```bash
# Через флаг -c
./shortener -c config.json

# Через флаг -config
./shortener -config config.json

# Через переменную окружения
CONFIG=config.json ./shortener
```

### Приоритет параметров

Конфигурация применяется в следующем порядке (от высшего к низшему):

1. Переменные окружения (наивысший приоритет)
2. Флаги командной строки
3. JSON файл конфигурации
4. Значения по умолчанию (наименьший приоритет)

Пример:

```bash
# config.json содержит: "server_address": "localhost:9090"
# Флаг -a устанавливает: localhost:8888
# Переменная окружения устанавливает: localhost:7777

SERVER_ADDRESS=localhost:7777 ./shortener -c config.json -a localhost:8888

# Результат: сервер запустится на localhost:7777 (переменная окружения)
```

### Примеры конфигураций

**Минимальная конфигурация:**
```json
{
    "server_address": "localhost:8080"
}
```

**Production с PostgreSQL:**
```json
{
    "server_address": ":8080",
    "base_url": "https://short.example.com",
    "database_dsn": "postgres://shortener:password@localhost:5432/shortener?sslmode=require",
    "enable_https": true
}
```

**Development с файловым хранилищем:**
```json
{
    "server_address": "localhost:3000",
    "base_url": "http://localhost:3000",
    "file_storage_path": "./data/urls.json",
    "enable_https": false
}
```

**С аудитом:**
```json
{
    "server_address": "localhost:8080",
    "base_url": "http://localhost:8080",
    "audit_file": "/var/log/shortener/audit.log",
    "audit_url": "http://audit-server:9000/events"
}
```