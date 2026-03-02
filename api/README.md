# API Documentation

В этой директории размещается документация API сервиса сокращения URL.

## Доступные API

### 1. HTTP REST API

HTTP API доступен на порту, указанном в конфигурации (по умолчанию `localhost:8080`).

**Основные эндпоинты:**
- `POST /` - сокращение URL (plain text)
- `POST /api/shorten` - сокращение URL (JSON)
- `POST /api/shorten/batch` - пакетное сокращение URL
- `GET /<id>` - получение оригинального URL (редирект)
- `GET /api/user/urls` - список URL пользователя
- `DELETE /api/user/urls` - удаление URL пользователя
- `GET /ping` - проверка доступности БД
- `GET /api/internal/stats` - статистика (требует trusted subnet)

**Документация:** См. основной README.md

### 2. gRPC API

gRPC API доступен на порту, указанном в конфигурации (по умолчанию `localhost:3200`).

**Proto файл:** `proto/shortener.proto`

**Сгенерированный код:** `internal/grpc/pb/`

**Доступные методы:**
- `ShortenURL` - сокращение URL
- `ExpandURL` - получение оригинального URL
- `ListUserURLs` - список URL пользователя

**Документация:** [GRPC.md](../GRPC.md)

## Генерация кода

### Protocol Buffers

Для генерации Go кода из proto файлов:

```bash
# Из корня проекта
./generate-proto.sh
```

Скрипт автоматически:
1. Проверит наличие protoc
2. Установит необходимые плагины (protoc-gen-go, protoc-gen-go-grpc)
3. Сгенерирует код в `internal/grpc/pb/`

### Требования

- Protocol Buffers compiler (protoc) версии 3.x или выше
- Go 1.21+

**Установка protoc:**
```bash
# macOS
brew install protobuf

# Linux (Ubuntu/Debian)
sudo apt-get install protobuf-compiler

# Linux (CentOS/RHEL)
sudo yum install protobuf-compiler
```

## Тестирование API

### HTTP API

```bash
# Сокращение URL
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'

# Получение оригинального URL
curl -L http://localhost:8080/abc123
```

### gRPC API

```bash
# Установка grpcurl
brew install grpcurl  # macOS
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest  # Другие ОС

# Сокращение URL
grpcurl -plaintext \
  -d '{"url": "https://example.com"}' \
  localhost:3200 shortener.ShortenerService/ShortenURL

# Список методов
grpcurl -plaintext localhost:3200 list shortener.ShortenerService
```

## Авторизация

Оба API используют одинаковую систему авторизации на основе подписанных токенов.

**HTTP:** Cookie `user_id` или header `Authorization: Bearer <token>`

**gRPC:** Metadata header `authorization: Bearer <token>`

При первом запросе без токена сервер автоматически создает нового пользователя и возвращает токен.

## Производительность

**HTTP:**
- Поддержка gzip сжатия
- Keep-alive соединения
- Graceful shutdown

**gRPC:**
- Бинарный протокол (Protocol Buffers)
- HTTP/2 мультиплексирование
- Меньший размер данных по сравнению с JSON

## Дополнительные ресурсы

- [Protocol Buffers Documentation](https://developers.google.com/protocol-buffers)
- [gRPC Documentation](https://grpc.io/docs/)
- [gRPC Go Tutorial](https://grpc.io/docs/languages/go/quickstart/)
- [grpcurl](https://github.com/fullstorydev/grpcurl)