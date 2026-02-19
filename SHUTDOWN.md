# Graceful Shutdown Guide

Руководство по корректному завершению работы сервиса сокращения URL.

## Обзор

Приложение поддерживает graceful shutdown - корректное завершение работы с обработкой всех активных запросов и сохранением всех данных.

## Поддерживаемые сигналы

Приложение корректно обрабатывает следующие сигналы завершения:

| Сигнал | Описание | Использование |
|--------|----------|---------------|
| `SIGINT` | Interrupt (Ctrl+C) | Остановка из терминала |
| `SIGTERM` | Terminate | Стандартный сигнал завершения (Docker, systemd) |
| `SIGQUIT` | Quit | Завершение с дампом состояния |

## Процесс завершения

При получении сигнала завершения приложение выполняет следующие шаги:

### 1. Получение сигнала

```
Получен сигнал interrupt. Завершение работы сервера...
```

### 2. Остановка фоновых операций

- Завершение всех асинхронных операций удаления URL
- Ожидание завершения активных задач в очереди
- Закрытие каналов для новых операций

### 3. Остановка HTTP сервера

- Прекращение приема новых соединений
- Ожидание завершения активных HTTP запросов (таймаут 30 секунд)
- Закрытие всех соединений

### 4. Сохранение данных

- Сохранение всех несохраненных данных в хранилище
- Закрытие файлов и соединений с базой данных
- Синхронизация логов

### 5. Завершение

```
Сервер остановлен
```

## Примеры использования

### Остановка через Ctrl+C

```bash
./shortener
# Нажмите Ctrl+C

^C
Получен сигнал interrupt. Завершение работы сервера...
Сервер остановлен
```

### Остановка через kill

```bash
# Запуск сервера
./shortener &
PID=$!

# Корректная остановка
kill -TERM $PID

# Или
kill -INT $PID

# Или
kill -QUIT $PID
```

### Остановка в Docker

```bash
# Docker автоматически отправляет SIGTERM
docker stop shortener-container

# С таймаутом (по умолчанию 10 секунд)
docker stop -t 60 shortener-container
```

### Остановка через systemd

```bash
# systemd отправляет SIGTERM
sudo systemctl stop shortener

# Проверка статуса
sudo systemctl status shortener
```

## Таймауты

### HTTP Server Shutdown

Таймаут для завершения активных HTTP запросов: **30 секунд**

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := server.Shutdown(ctx); err != nil {
    log.Printf("Ошибка при завершении сервера: %v", err)
}
```

Если запросы не завершатся за 30 секунд, сервер принудительно закроется.

### Фоновые операции

Фоновые операции (удаление URL) завершаются через context cancellation:

```go
func (s *URLService) Shutdown() {
    s.cancel()      // Отменяем context
    s.wg.Wait()     // Ждем завершения всех горутин
}
```

## Гарантии

### Обработка запросов

✅ Все активные HTTP запросы будут обработаны до конца (в пределах таймаута)

✅ Новые запросы не принимаются после получения сигнала

✅ Клиенты получат корректные ответы на свои запросы

### Сохранение данных

✅ Все несохраненные URL будут записаны в хранилище

✅ Файловое хранилище корректно закроет файл

✅ Соединения с PostgreSQL будут корректно закрыты

✅ Все операции удаления в очереди будут завершены

### Логирование

✅ Все логи будут синхронизированы перед завершением

✅ События аудита будут записаны

## Тестирование

### Ручное тестирование

```bash
# Терминал 1: Запуск сервера
./shortener

# Терминал 2: Создание запроса
curl -X POST http://localhost:8080/ -d "https://example.com" &

# Терминал 1: Остановка (Ctrl+C)
# Проверьте, что запрос завершился успешно
```

### Автоматическое тестирование

```bash
go test -v ./cmd/shortener -run TestGracefulShutdown
```

### Нагрузочное тестирование

```bash
# Запуск сервера
./shortener &
PID=$!

# Генерация нагрузки
for i in {1..100}; do
    curl -X POST http://localhost:8080/ -d "https://example.com/$i" &
done

# Остановка во время нагрузки
sleep 1
kill -TERM $PID

# Проверка, что все запросы обработаны
wait
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

# Важно: используйте exec форму CMD для корректной обработки сигналов
CMD ["./shortener"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  shortener:
    build: .
    ports:
      - "8080:8080"
    # Таймаут для graceful shutdown (по умолчанию 10 секунд)
    stop_grace_period: 60s
    environment:
      - DATABASE_DSN=postgres://user:pass@db:5432/shortener
```

### Остановка контейнера

```bash
# Graceful shutdown с таймаутом
docker-compose down

# С увеличенным таймаутом
docker-compose down -t 60
```

## Kubernetes

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: shortener
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: shortener
        image: shortener:latest
        ports:
        - containerPort: 8080
        # Graceful shutdown период
        terminationGracePeriodSeconds: 60
        lifecycle:
          preStop:
            exec:
              # Опционально: дополнительная задержка перед SIGTERM
              command: ["/bin/sh", "-c", "sleep 5"]
```

### Rolling Update

```bash
# Kubernetes автоматически выполняет graceful shutdown
kubectl rollout restart deployment/shortener

# Проверка статуса
kubectl rollout status deployment/shortener
```

## Systemd

### Service файл

```ini
[Unit]
Description=URL Shortener Service
After=network.target

[Service]
Type=simple
User=shortener
WorkingDirectory=/opt/shortener
ExecStart=/opt/shortener/shortener
Restart=on-failure
RestartSec=5

# Graceful shutdown настройки
KillMode=mixed
KillSignal=SIGTERM
TimeoutStopSec=60

[Install]
WantedBy=multi-user.target
```

### Управление сервисом

```bash
# Запуск
sudo systemctl start shortener

# Остановка (отправляет SIGTERM)
sudo systemctl stop shortener

# Перезапуск (graceful)
sudo systemctl restart shortener

# Проверка логов
sudo journalctl -u shortener -f
```

## Мониторинг

### Логи при завершении

Нормальное завершение:
```
Получен сигнал terminated. Завершение работы сервера...
Сервер остановлен
```

Завершение с ошибкой:
```
Получен сигнал terminated. Завершение работы сервера...
Ошибка при завершении сервера: context deadline exceeded
Сервер остановлен
```

### Метрики

Рекомендуется отслеживать:
- Время завершения работы
- Количество активных запросов при получении сигнала
- Количество несохраненных данных
- Ошибки при завершении

## Troubleshooting

### Проблема: Сервер не останавливается

**Причина:** Активные запросы не завершаются в течение таймаута

**Решение:**
1. Увеличьте таймаут в коде:
```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
```

2. Проверьте, нет ли зависших запросов:
```bash
# Проверка активных соединений
netstat -an | grep :8080 | grep ESTABLISHED
```

### Проблема: Данные не сохраняются

**Причина:** Хранилище не успевает сохранить данные

**Решение:**
1. Проверьте логи на ошибки сохранения
2. Убедитесь, что `defer store.Close()` вызывается
3. Проверьте права доступа к файлу хранилища

### Проблема: Docker контейнер убивается принудительно

**Причина:** Таймаут Docker меньше времени завершения

**Решение:**
Увеличьте `stop_grace_period` в docker-compose.yml:
```yaml
services:
  shortener:
    stop_grace_period: 90s
```

### Проблема: Kubernetes pod в состоянии Terminating

**Причина:** Pod не завершается в течение `terminationGracePeriodSeconds`

**Решение:**
1. Увеличьте таймаут:
```yaml
terminationGracePeriodSeconds: 90
```

2. Проверьте логи pod:
```bash
kubectl logs pod-name
```

## Best Practices

### 1. Используйте достаточные таймауты

```go
// Минимум 30 секунд для обработки запросов
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
```

### 2. Логируйте процесс завершения

```go
log.Printf("Получен сигнал %v. Завершение работы...", sig)
log.Println("Остановка фоновых операций...")
log.Println("Остановка HTTP сервера...")
log.Println("Сохранение данных...")
log.Println("Сервер остановлен")
```

### 3. Тестируйте graceful shutdown

```bash
# Регулярно тестируйте под нагрузкой
go test -v ./cmd/shortener -run TestGracefulShutdown
```

### 4. Мониторьте метрики завершения

- Время завершения
- Количество потерянных запросов
- Ошибки при сохранении данных

### 5. Документируйте процесс

Убедитесь, что команда знает:
- Какие сигналы использовать
- Сколько времени занимает завершение
- Что происходит с данными

## Дополнительные ресурсы

- [Go HTTP Server Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)
- [Docker Stop](https://docs.docker.com/engine/reference/commandline/stop/)
- [Kubernetes Pod Lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)
- [Systemd Service Management](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
