# Configuration Guide

Руководство по конфигурации сервиса сокращения URL.

## Способы конфигурации

Приложение поддерживает три способа конфигурации:

1. **JSON файл** - удобно для production и разных окружений
2. **Флаги командной строки** - быстрая настройка при запуске
3. **Переменные окружения** - интеграция с Docker, Kubernetes и т.д.

## Приоритет параметров

Параметры применяются в следующем порядке (от высшего к низшему):

```
Переменные окружения > Флаги > JSON файл > Значения по умолчанию
```

Это означает, что переменная окружения всегда переопределит значение из флага или JSON файла.

## JSON конфигурация

### Формат файла

```json
{
    "server_address": "localhost:8080",
    "base_url": "http://localhost:8080",
    "file_storage_path": "/tmp/short-url-db.json",
    "database_dsn": "",
    "enable_https": false,
    "audit_file": "",
    "audit_url": ""
}
```

### Описание полей

| Поле | Тип | Описание | По умолчанию |
|------|-----|----------|--------------|
| `server_address` | string | Адрес и порт для запуска сервера | `localhost:8080` |
| `base_url` | string | Базовый URL для сокращенных ссылок | `http://localhost:8080` |
| `file_storage_path` | string | Путь к файлу для хранения данных | `/tmp/short-url-db.json` |
| `database_dsn` | string | Строка подключения к PostgreSQL | `""` (не используется) |
| `enable_https` | boolean | Включить HTTPS | `false` |
| `audit_file` | string | Путь к файлу аудита | `""` (не используется) |
| `audit_url` | string | URL сервера аудита | `""` (не используется) |

### Использование

```bash
# Через флаг -c
./shortener -c config.json

# Через флаг -config (альтернатива)
./shortener -config /etc/shortener/config.json

# Через переменную окружения CONFIG
CONFIG=config.json ./shortener

# Переменная окружения имеет приоритет над флагом
CONFIG=prod.json ./shortener -c dev.json
# Будет использован prod.json
```

### Примеры конфигураций

#### Development

```json
{
    "server_address": "localhost:3000",
    "base_url": "http://localhost:3000",
    "file_storage_path": "./data/urls.json"
}
```

Запуск:
```bash
./shortener -c config.dev.json
```

#### Staging

```json
{
    "server_address": ":8080",
    "base_url": "https://staging.short.example.com",
    "database_dsn": "postgres://user:pass@db-staging:5432/shortener",
    "enable_https": true
}
```

Запуск:
```bash
./shortener -c config.staging.json
```

#### Production

```json
{
    "server_address": ":443",
    "base_url": "https://short.example.com",
    "database_dsn": "postgres://user:pass@db-prod:5432/shortener?sslmode=require",
    "enable_https": true,
    "audit_file": "/var/log/shortener/audit.log"
}
```

Запуск:
```bash
./shortener -c config.production.json
```

## Флаги командной строки

### Список флагов

```bash
./shortener -h
```

| Флаг | Короткий | Описание |
|------|----------|----------|
| `-config` | `-c` | Путь к JSON файлу конфигурации |
| `-a` | | Адрес запуска HTTP-сервера |
| `-b` | | Базовый адрес результирующего сокращённого URL |
| `-f` | | Путь до файла с данными |
| `-d` | | Строка подключения к базе данных |
| `-s` | | Включить HTTPS |
| `-audit-file` | | Путь к файлу аудита |
| `-audit-url` | | URL удаленного сервера аудита |

### Примеры использования

```bash
# Базовый запуск
./shortener -a localhost:8080 -b http://localhost:8080

# С базой данных
./shortener -d "postgres://user:pass@localhost/shortener"

# С HTTPS
./shortener -s -a localhost:8443 -b https://localhost:8443

# Все параметры
./shortener \
  -a :8080 \
  -b https://short.example.com \
  -d "postgres://user:pass@localhost/shortener" \
  -s \
  -audit-file /var/log/audit.log
```

## Переменные окружения

### Список переменных

| Переменная | Описание | Пример |
|------------|----------|--------|
| `CONFIG` | Путь к JSON файлу конфигурации | `config.json` |
| `SERVER_ADDRESS` | Адрес запуска HTTP-сервера | `localhost:8080` |
| `BASE_URL` | Базовый адрес результирующего сокращённого URL | `http://localhost:8080` |
| `FILE_STORAGE_PATH` | Путь до файла с данными | `/tmp/short-url-db.json` |
| `DATABASE_DSN` | Строка подключения к базе данных | `postgres://...` |
| `ENABLE_HTTPS` | Включить HTTPS | `true` или `false` |
| `AUDIT_FILE` | Путь к файлу аудита | `/var/log/audit.log` |
| `AUDIT_URL` | URL удаленного сервера аудита | `http://audit:9000` |

### Примеры использования

```bash
# Базовый запуск
SERVER_ADDRESS=localhost:8080 \
BASE_URL=http://localhost:8080 \
./shortener

# С базой данных
DATABASE_DSN="postgres://user:pass@localhost/shortener" \
./shortener

# С HTTPS
ENABLE_HTTPS=true \
SERVER_ADDRESS=localhost:8443 \
BASE_URL=https://localhost:8443 \
./shortener

# Через .env файл
cat > .env << EOF
SERVER_ADDRESS=localhost:8080
BASE_URL=http://localhost:8080
DATABASE_DSN=postgres://user:pass@localhost/shortener
ENABLE_HTTPS=false
EOF

# Загрузка и запуск
export $(cat .env | xargs)
./shortener
```

## Комбинирование способов

### Пример 1: JSON + переменные окружения

```json
// config.json
{
    "server_address": "localhost:8080",
    "base_url": "http://localhost:8080",
    "file_storage_path": "/tmp/urls.json"
}
```

```bash
# Переопределяем database_dsn через переменную окружения
DATABASE_DSN="postgres://user:pass@localhost/shortener" \
./shortener -c config.json

# Результат:
# - server_address: localhost:8080 (из JSON)
# - base_url: http://localhost:8080 (из JSON)
# - file_storage_path: /tmp/urls.json (из JSON)
# - database_dsn: postgres://... (из переменной окружения)
```

### Пример 2: JSON + флаги + переменные окружения

```json
// config.json
{
    "server_address": "localhost:9090",
    "base_url": "http://localhost:9090"
}
```

```bash
# Комбинация всех способов
SERVER_ADDRESS=localhost:7777 \
./shortener -c config.json -a localhost:8888

# Результат:
# - server_address: localhost:7777 (переменная окружения - наивысший приоритет)
# - base_url: http://localhost:9090 (из JSON)
```

### Пример 3: Разные окружения

```bash
# Development
CONFIG=config.dev.json ./shortener

# Staging с переопределением базы данных
CONFIG=config.staging.json \
DATABASE_DSN="postgres://user:pass@staging-db/shortener" \
./shortener

# Production
CONFIG=config.prod.json \
DATABASE_DSN="$PROD_DATABASE_DSN" \
./shortener
```

## Docker

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o shortener ./cmd/shortener

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/shortener .
COPY config.json .

EXPOSE 8080
CMD ["./shortener", "-c", "config.json"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  shortener:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_DSN=postgres://user:pass@db:5432/shortener
      - ENABLE_HTTPS=false
    volumes:
      - ./config.json:/root/config.json:ro
    depends_on:
      - db

  db:
    image: postgres:15
    environment:
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
      - POSTGRES_DB=shortener
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

Запуск:
```bash
docker-compose up
```

## Kubernetes

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: shortener-config
data:
  config.json: |
    {
      "server_address": ":8080",
      "base_url": "https://short.example.com",
      "enable_https": true
    }
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: shortener
spec:
  replicas: 3
  selector:
    matchLabels:
      app: shortener
  template:
    metadata:
      labels:
        app: shortener
    spec:
      containers:
      - name: shortener
        image: shortener:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_DSN
          valueFrom:
            secretKeyRef:
              name: shortener-secrets
              key: database-dsn
        volumeMounts:
        - name: config
          mountPath: /etc/shortener
          readOnly: true
        args:
        - "-c"
        - "/etc/shortener/config.json"
      volumes:
      - name: config
        configMap:
          name: shortener-config
```

## Валидация конфигурации

Приложение автоматически валидирует конфигурацию при запуске:

### Обязательные поля

- `server_address` - не может быть пустым
- `base_url` - должен содержать схему (http:// или https://) и хост

### Примеры ошибок

```bash
# Пустой server_address
Error: ошибка валидации конфигурации: адрес сервера не может быть пустым

# Некорректный base_url
Error: ошибка валидации конфигурации: базовый URL должен содержать схему (http:// или https://)

# Невалидный JSON
Error: ошибка загрузки конфигурации из файла: не удалось распарсить JSON: invalid character...

# Несуществующий файл
Error: ошибка загрузки конфигурации из файла: не удалось прочитать файл: no such file or directory
```

## Best Practices

### 1. Используйте JSON для базовой конфигурации

```json
{
    "server_address": ":8080",
    "base_url": "https://short.example.com",
    "enable_https": true
}
```

### 2. Секреты через переменные окружения

```bash
# Не храните пароли в JSON!
DATABASE_DSN="postgres://user:$DB_PASSWORD@localhost/shortener" \
./shortener -c config.json
```

### 3. Разные конфигурации для разных окружений

```
config.dev.json
config.staging.json
config.production.json
```

### 4. Используйте .gitignore

```gitignore
# Не коммитьте конфигурации с секретами
config.json
config.*.json
!config.json.example
```

### 5. Документируйте конфигурацию

Создайте `config.json.example`:

```json
{
    "server_address": "localhost:8080",
    "base_url": "http://localhost:8080",
    "file_storage_path": "/tmp/short-url-db.json",
    "database_dsn": "postgres://user:pass@localhost/shortener",
    "enable_https": false
}
```

## Troubleshooting

### Проблема: Конфигурация не применяется

**Решение:** Проверьте приоритет параметров. Переменные окружения имеют наивысший приоритет.

```bash
# Проверьте переменные окружения
env | grep -E "(SERVER_ADDRESS|BASE_URL|DATABASE_DSN|ENABLE_HTTPS|CONFIG)"

# Очистите переменные если нужно
unset SERVER_ADDRESS BASE_URL DATABASE_DSN ENABLE_HTTPS CONFIG
```

### Проблема: Ошибка парсинга JSON

**Решение:** Проверьте синтаксис JSON:

```bash
# Валидация JSON
cat config.json | jq .

# Или используйте онлайн валидатор
# https://jsonlint.com/
```

### Проблема: Файл конфигурации не найден

**Решение:** Используйте абсолютный путь или проверьте рабочую директорию:

```bash
# Абсолютный путь
./shortener -c /etc/shortener/config.json

# Относительный путь от текущей директории
./shortener -c ./config.json

# Проверка рабочей директории
pwd
ls -la config.json
```

## Примеры для разных сценариев

### Локальная разработка

```bash
./shortener -a localhost:3000 -b http://localhost:3000
```

### Тестирование с PostgreSQL

```bash
DATABASE_DSN="postgres://test:test@localhost/test_shortener" \
./shortener -c config.test.json
```

### Production с мониторингом

```json
{
    "server_address": ":8080",
    "base_url": "https://short.example.com",
    "database_dsn": "postgres://user:pass@db:5432/shortener?sslmode=require",
    "enable_https": true,
    "audit_file": "/var/log/shortener/audit.log",
    "audit_url": "http://monitoring:9000/events"
}
```

### Микросервисная архитектура

```yaml
# docker-compose.yml
services:
  shortener-api:
    image: shortener:latest
    environment:
      - CONFIG=/config/api.json
    volumes:
      - ./configs:/config:ro

  shortener-worker:
    image: shortener:latest
    environment:
      - CONFIG=/config/worker.json
    volumes:
      - ./configs:/config:ro
```

## Дополнительные ресурсы

- [cmd/shortener/README.md](cmd/shortener/README.md) - Документация по сборке и запуску
- [HTTPS.md](HTTPS.md) - Руководство по настройке HTTPS
- [DEVELOPMENT.md](DEVELOPMENT.md) - Руководство разработчика
