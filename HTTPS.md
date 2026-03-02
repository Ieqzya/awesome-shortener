# HTTPS Configuration Guide

Это руководство описывает настройку и использование HTTPS в сервисе сокращения URL.

## Быстрый старт

### 1. Генерация сертификата

Для разработки и тестирования используйте самоподписанный сертификат:

```bash
# Используйте готовый скрипт
./generate-cert.sh

# Или создайте вручную
openssl req -x509 -newkey rsa:2048 \
  -keyout cmd/shortener/key.pem \
  -out cmd/shortener/cert.pem \
  -days 365 \
  -nodes \
  -subj "/CN=localhost"
```

### 2. Запуск сервера

```bash
# Сборка приложения
go build -o shortener ./cmd/shortener

# Запуск с HTTPS через флаг
./shortener -s

# Или через переменную окружения
ENABLE_HTTPS=true ./shortener
```

### 3. Проверка работы

```bash
# Проверка доступности (игнорируем самоподписанный сертификат)
curl -k https://localhost:8080/ping

# Создание короткой ссылки
curl -k -X POST https://localhost:8080/ \
  -H "Content-Type: text/plain" \
  -d "https://example.com"

# Ответ: https://localhost:8080/abc123
```

## Конфигурация

### Флаги командной строки

```bash
# Только HTTPS
./shortener -s

# HTTPS с кастомным портом
./shortener -s -a localhost:8443

# HTTPS с кастомным базовым URL
./shortener -s -a localhost:8443 -b https://localhost:8443

# Все параметры вместе
./shortener -s \
  -a localhost:8443 \
  -b https://localhost:8443 \
  -d "postgres://user:pass@localhost/db"
```

### Переменные окружения

```bash
# Базовая конфигурация
export ENABLE_HTTPS=true
export SERVER_ADDRESS=localhost:8443
export BASE_URL=https://localhost:8443
./shortener

# С базой данных
export ENABLE_HTTPS=true
export DATABASE_DSN="postgres://user:pass@localhost/shortener"
./shortener
```

### Приоритет параметров

Переменные окружения имеют наивысший приоритет:

1. Переменные окружения (наивысший)
2. Флаги командной строки
3. Значения по умолчанию (наименьший)

Пример:
```bash
# Флаг -s будет переопределен переменной окружения
ENABLE_HTTPS=false ./shortener -s
# Результат: HTTP (не HTTPS)
```

## Production deployment

### Использование Let's Encrypt

Для production рекомендуется использовать сертификаты от Let's Encrypt:

```bash
# Установка certbot
sudo apt-get install certbot

# Получение сертификата
sudo certbot certonly --standalone -d yourdomain.com

# Копирование сертификатов
sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem cmd/shortener/cert.pem
sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem cmd/shortener/key.pem
sudo chown $USER:$USER cmd/shortener/*.pem

# Запуск
./shortener -s -a :443 -b https://yourdomain.com
```

### Автоматическое обновление сертификатов

Создайте скрипт для автоматического обновления:

```bash
#!/bin/bash
# renew-cert.sh

certbot renew --quiet
cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem /path/to/cmd/shortener/cert.pem
cp /etc/letsencrypt/live/yourdomain.com/privkey.pem /path/to/cmd/shortener/key.pem
systemctl restart shortener
```

Добавьте в crontab:
```
0 0 * * 0 /path/to/renew-cert.sh
```

### Systemd service

Создайте файл `/etc/systemd/system/shortener.service`:

```ini
[Unit]
Description=URL Shortener Service
After=network.target

[Service]
Type=simple
User=shortener
WorkingDirectory=/opt/shortener
Environment="ENABLE_HTTPS=true"
Environment="SERVER_ADDRESS=:443"
Environment="BASE_URL=https://yourdomain.com"
Environment="DATABASE_DSN=postgres://user:pass@localhost/shortener"
ExecStart=/opt/shortener/shortener
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Запуск:
```bash
sudo systemctl daemon-reload
sudo systemctl enable shortener
sudo systemctl start shortener
sudo systemctl status shortener
```

## Docker

### Dockerfile с HTTPS

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o shortener ./cmd/shortener

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/shortener .
COPY cmd/shortener/cert.pem .
COPY cmd/shortener/key.pem .

EXPOSE 8443

CMD ["./shortener", "-s", "-a", ":8443"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  shortener:
    build: .
    ports:
      - "8443:8443"
    environment:
      - ENABLE_HTTPS=true
      - SERVER_ADDRESS=:8443
      - BASE_URL=https://localhost:8443
      - DATABASE_DSN=postgres://user:pass@db:5432/shortener
    volumes:
      - ./cmd/shortener/cert.pem:/root/cert.pem:ro
      - ./cmd/shortener/key.pem:/root/key.pem:ro
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

## Безопасность

### Рекомендации

1. **Никогда не коммитьте сертификаты в git**
   - Файлы `*.pem`, `*.crt`, `*.key` добавлены в `.gitignore`

2. **Используйте сильные ключи**
   - Минимум RSA 2048 бит
   - Рекомендуется RSA 4096 бит или ECDSA

3. **Регулярно обновляйте сертификаты**
   - Let's Encrypt сертификаты действительны 90 дней
   - Настройте автоматическое обновление

4. **Ограничьте доступ к ключам**
   ```bash
   chmod 600 cmd/shortener/key.pem
   chmod 644 cmd/shortener/cert.pem
   ```

5. **Используйте HTTPS для базового URL**
   ```bash
   # Правильно
   ./shortener -s -b https://yourdomain.com
   
   # Неправильно (смешанный контент)
   ./shortener -s -b http://yourdomain.com
   ```

## Troubleshooting

### Ошибка: "certificate signed by unknown authority"

Это нормально для самоподписанных сертификатов. Используйте флаг `-k` в curl:
```bash
curl -k https://localhost:8080/
```

### Ошибка: "bind: permission denied" на порту 443

Порты < 1024 требуют root привилегий:
```bash
# Вариант 1: Используйте sudo
sudo ./shortener -s -a :443

# Вариант 2: Дайте capability
sudo setcap 'cap_net_bind_service=+ep' ./shortener
./shortener -s -a :443

# Вариант 3: Используйте порт > 1024
./shortener -s -a :8443
```

### Ошибка: "no such file or directory: cert.pem"

Убедитесь, что сертификаты находятся в правильной директории:
```bash
ls -la cmd/shortener/*.pem
```

Если файлов нет, сгенерируйте их:
```bash
./generate-cert.sh
```

### Проверка сертификата

```bash
# Информация о сертификате
openssl x509 -in cmd/shortener/cert.pem -text -noout

# Проверка соответствия ключа и сертификата
openssl x509 -noout -modulus -in cmd/shortener/cert.pem | openssl md5
openssl rsa -noout -modulus -in cmd/shortener/key.pem | openssl md5
# Хеши должны совпадать
```

## Тестирование

### Локальное тестирование

```bash
# Запуск сервера
./shortener -s &
SERVER_PID=$!

# Тесты
curl -k https://localhost:8080/ping
curl -k -X POST https://localhost:8080/ -d "https://example.com"

# Остановка
kill $SERVER_PID
```

### Интеграционные тесты

```go
func TestHTTPS(t *testing.T) {
    // Создаем тестовый сервер с HTTPS
    server := httptest.NewTLSServer(handler)
    defer server.Close()
    
    // Клиент, игнорирующий проверку сертификата
    client := server.Client()
    
    resp, err := client.Get(server.URL)
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected 200, got %d", resp.StatusCode)
    }
}
```

## Дополнительные ресурсы

- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
- [SSL Labs Server Test](https://www.ssllabs.com/ssltest/)
- [Go TLS Documentation](https://pkg.go.dev/crypto/tls)
