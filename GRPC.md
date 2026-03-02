# gRPC API Documentation

Руководство по использованию gRPC API сервиса сокращения URL.

## Установка protoc

Для генерации Go кода из proto файлов необходим Protocol Buffers compiler (protoc).

### macOS

```bash
brew install protobuf
```

### Linux (Ubuntu/Debian)

```bash
sudo apt-get update
sudo apt-get install -y protobuf-compiler
```

### Linux (CentOS/RHEL)

```bash
sudo yum install -y protobuf-compiler
```

### Windows

Скачайте pre-compiled binary:
https://github.com/protocolbuffers/protobuf/releases

Распакуйте и добавьте `bin/protoc.exe` в PATH.

### Проверка установки

```bash
protoc --version
# Должно вывести: libprotoc 3.x.x или выше
```

## Генерация Go кода

После установки protoc запустите:

```bash
./generate-proto.sh
```

Скрипт автоматически:
1. Проверит наличие protoc
2. Установит protoc-gen-go и protoc-gen-go-grpc если нужно
3. Сгенерирует Go код в `internal/grpc/pb/`

## API Endpoints

### 1. ShortenURL - Сокращение URL

**Запрос:**
```protobuf
message URLShortenRequest {
  string url = 1;
}
```

**Ответ:**
```protobuf
message URLShortenResponse {
  string result = 1;  // Сокращенный URL
}
```

**Пример (grpcurl):**
```bash
grpcurl -plaintext \
  -d '{"url": "https://example.com"}' \
  localhost:3200 shortener.ShortenerService/ShortenURL
```

**Ответ:**
```json
{
  "result": "http://localhost:8080/abc123"
}
```

### 2. ExpandURL - Получение оригинального URL

**Запрос:**
```protobuf
message URLExpandRequest {
  string id = 1;  // Короткий ID
}
```

**Ответ:**
```protobuf
message URLExpandResponse {
  string result = 1;  // Оригинальный URL
}
```

**Пример:**
```bash
grpcurl -plaintext \
  -d '{"id": "abc123"}' \
  localhost:3200 shortener.ShortenerService/ExpandURL
```

**Ответ:**
```json
{
  "result": "https://example.com"
}
```

### 3. ListUserURLs - Список URL пользователя

**Запрос:**
```protobuf
google.protobuf.Empty
```

**Ответ:**
```protobuf
message UserURLsResponse {
  repeated URLData url = 1;
}

message URLData {
  string short_url = 1;
  string original_url = 2;
}
```

**Пример:**
```bash
grpcurl -plaintext \
  -H "authorization: Bearer <user_token>" \
  localhost:3200 shortener.ShortenerService/ListUserURLs
```

**Ответ:**
```json
{
  "url": [
    {
      "short_url": "http://localhost:8080/abc123",
      "original_url": "https://example.com"
    },
    {
      "short_url": "http://localhost:8080/def456",
      "original_url": "https://google.com"
    }
  ]
}
```

## Авторизация

gRPC API использует metadata для передачи авторизационных данных.

### Формат

```
authorization: Bearer <signed_user_id>
```

### Получение токена

При первом запросе без authorization header сервер автоматически создает нового пользователя и возвращает токен в response metadata.

### Использование токена

Передавайте полученный токен в последующих запросах:

```bash
grpcurl -plaintext \
  -H "authorization: Bearer eyJhbGc..." \
  -d '{"url": "https://example.com"}' \
  localhost:3200 shortener.ShortenerService/ShortenURL
```

## Конфигурация

gRPC сервер использует те же параметры конфигурации, что и HTTP сервер:

```json
{
  "server_address": "localhost:8080",
  "grpc_address": "localhost:3200",
  "base_url": "http://localhost:8080",
  "enable_https": false
}
```

### Переменные окружения

```bash
GRPC_ADDRESS=localhost:3200  # Адрес gRPC сервера
```

### Флаги

```bash
./shortener -grpc-address localhost:3200
```

## Примеры клиентов

### Go Client

```go
package main

import (
    "context"
    "log"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "awesome-shortener/internal/grpc/pb"
)

func main() {
    // Подключение к серверу
    conn, err := grpc.Dial("localhost:3200", 
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    client := pb.NewShortenerServiceClient(conn)
    
    // Сокращение URL
    resp, err := client.ShortenURL(context.Background(), &pb.URLShortenRequest{
        Url: "https://example.com",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Shortened URL: %s", resp.Result)
}
```

### Python Client

```python
import grpc
import shortener_pb2
import shortener_pb2_grpc

# Подключение
channel = grpc.insecure_channel('localhost:3200')
stub = shortener_pb2_grpc.ShortenerServiceStub(channel)

# Сокращение URL
request = shortener_pb2.URLShortenRequest(url='https://example.com')
response = stub.ShortenURL(request)
print(f'Shortened URL: {response.result}')
```

## Тестирование

### Установка grpcurl

```bash
# macOS
brew install grpcurl

# Linux
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Windows
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

### Список доступных методов

```bash
grpcurl -plaintext localhost:3200 list
grpcurl -plaintext localhost:3200 list shortener.ShortenerService
```

### Описание метода

```bash
grpcurl -plaintext localhost:3200 describe shortener.ShortenerService.ShortenURL
```

### Reflection

Сервер поддерживает gRPC reflection для автоматического обнаружения API:

```bash
grpcurl -plaintext localhost:3200 describe
```

## Производительность

gRPC обеспечивает:
- Бинарный протокол (Protocol Buffers) - меньше размер данных
- HTTP/2 - мультиплексирование, server push
- Streaming - поддержка потоковой передачи данных

### Бенчмарки

```bash
# HTTP
ab -n 10000 -c 100 http://localhost:8080/

# gRPC
ghz --insecure \
  --proto proto/shortener.proto \
  --call shortener.ShortenerService/ShortenURL \
  -d '{"url":"https://example.com"}' \
  -n 10000 -c 100 \
  localhost:3200
```

## Безопасность

### TLS/SSL

Для production используйте TLS:

```go
creds, err := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
if err != nil {
    log.Fatal(err)
}

server := grpc.NewServer(grpc.Creds(creds))
```

Клиент:

```go
creds, err := credentials.NewClientTLSFromFile("cert.pem", "")
if err != nil {
    log.Fatal(err)
}

conn, err := grpc.Dial("localhost:3200", grpc.WithTransportCredentials(creds))
```

### Аутентификация

Используйте interceptors для проверки токенов:

```go
func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
    }
    
    // Проверка токена
    // ...
    
    return handler(ctx, req)
}

server := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor))
```

## Troubleshooting

### Ошибка: protoc not found

Установите protoc (см. раздел "Установка protoc")

### Ошибка: protoc-gen-go not found

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Убедитесь, что $GOPATH/bin в PATH
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Ошибка: connection refused

Проверьте, что gRPC сервер запущен:

```bash
netstat -an | grep 3200
```

### Ошибка: Unimplemented

Убедитесь, что сгенерированный код актуален:

```bash
./generate-proto.sh
go build ./...
```

## Дополнительные ресурсы

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers](https://developers.google.com/protocol-buffers)
- [gRPC Go Tutorial](https://grpc.io/docs/languages/go/quickstart/)
- [grpcurl](https://github.com/fullstorydev/grpcurl)
