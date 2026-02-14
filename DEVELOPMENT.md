# Руководство разработчика

Этот документ содержит информацию о инструментах разработки и кодогенерации проекта awesome-shortener.

## Инструменты

### 1. Staticlint - Multichecker для статического анализа

Комплексный инструмент статического анализа, объединяющий множество анализаторов.

**Расположение:** `cmd/staticlint/`

**Запуск:**
```bash
# Сборка
go build -o staticlint ./cmd/staticlint

# Проверка всего проекта
./staticlint ./...

# Проверка конкретного пакета
./staticlint ./internal/handler
```

**Включенные анализаторы:**
- 21 стандартный анализатор из golang.org/x/tools
- Все анализаторы класса SA из staticcheck.io
- Дополнительные анализаторы: S1000, ST1000, QF1001
- Публичные анализаторы: errcheck, bodyclose
- Собственный анализатор: osexit (запрещает os.Exit в main)

**Документация:** [cmd/staticlint/README.md](cmd/staticlint/README.md)

### 2. Reset Generator - Генератор методов Reset()

Автоматически генерирует методы Reset() для структур с комментарием `// generate:reset`.

**Расположение:** `cmd/reset/`

**Запуск:**
```bash
go run cmd/reset/main.go
```

**Использование:**
```go
// generate:reset
type MyStruct struct {
    ID   int
    Name string
    Tags []string
}

// После генерации доступен метод:
func (m *MyStruct) Reset() {
    // Автоматически сгенерированный код
}
```

**Правила сброса:**
- Примитивы → нулевые значения
- Слайсы → `slice[:0]` (сохраняется capacity)
- Мапы → `clear(map)`
- Вложенные структуры → вызов Reset() если есть
- Указатели → сброс значения если не nil

**Документация:** [cmd/reset/README.md](cmd/reset/README.md)

### 3. Pool - Типобезопасный пул объектов

Generic пул объектов для эффективного переиспользования структур с методом Reset().

**Расположение:** `internal/pool/`

**Использование:**
```go
import "awesome-shortener/internal/pool"

// Создаем пул для структур с методом Reset()
p := pool.New(func() *MyStruct {
    return &MyStruct{
        Buffer: make([]byte, 0, 1024),
    }
})

// Получаем объект из пула
obj := p.Get()
defer p.Put(obj) // Автоматически вызовет Reset()

// Используем объект
obj.ID = 123
```

**Преимущества:**
- Снижение нагрузки на GC (~194x быстрее создания новых объектов)
- Типобезопасность на уровне компиляции (generics)
- Автоматический вызов Reset() перед возвратом в пул
- Потокобезопасность

**Документация:** [internal/pool/README.md](internal/pool/README.md)

## Архитектурные улучшения

### Dependency Injection

Проект использует dependency injection для слабой связанности компонентов:

```go
// AuthService вместо глобальных переменных
authService := auth.NewAuthService()

// App с внедренными зависимостями
app := handler.NewApp(cfg, store)
```

### Безопасность

- Генерация ID использует только `crypto/rand` (без fallback на `math/rand`)
- Все методы проверяют receiver на nil
- Указатели проверяются перед разыменованием

### Производительность

- Слайсы сбрасываются через `[:0]` для сохранения capacity
- Мапы очищаются через встроенную функцию `clear()`
- Минимальные аллокации памяти

## Workflow разработки

### 1. Добавление новой структуры с Reset

```go
// 1. Добавьте комментарий
// generate:reset
type NewStruct struct {
    Field1 int
    Field2 string
}

// 2. Запустите генератор
go run cmd/reset/main.go

// 3. Проверьте сгенерированный код
cat reset.gen.go
```

### 2. Проверка кода перед коммитом

```bash
# Форматирование
gofmt -w -s .
goimports -w .

# Статический анализ
go build -o staticlint ./cmd/staticlint
./staticlint ./...

# Тесты
go test ./...

# Покрытие
go test -cover ./...
```

### 3. Интеграция в CI/CD

```yaml
# .github/workflows/ci.yml
- name: Generate Reset methods
  run: go run cmd/reset/main.go

- name: Check generated files
  run: |
    if [[ -n $(git status --porcelain) ]]; then
      echo "Generated files are not up to date"
      exit 1
    fi

- name: Static analysis
  run: |
    go build -o staticlint ./cmd/staticlint
    ./staticlint ./...

- name: Tests
  run: go test -v -race -coverprofile=coverage.out ./...
```

## Makefile

Добавьте в Makefile для удобства:

```makefile
.PHONY: generate
generate:
	go run cmd/reset/main.go

.PHONY: lint
lint:
	go build -o staticlint ./cmd/staticlint
	./staticlint ./...

.PHONY: test
test:
	go test -v -race ./...

.PHONY: coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: check
check: generate lint test
	@echo "All checks passed!"
```

## Godoc документация

Проект полностью документирован в формате godoc:

```bash
# Локальный просмотр документации
godoc -http=:6060

# Открыть в браузере
open http://localhost:6060/pkg/awesome-shortener/
```

**Документированные пакеты:**
- `cmd/staticlint` - multichecker
- `cmd/staticlint/osexit` - анализатор os.Exit
- `cmd/reset` - генератор Reset
- `internal/auth` - аутентификация
- `internal/config` - конфигурация
- `internal/handler` - HTTP обработчики
- `internal/middleware` - HTTP middleware
- `internal/model` - модели данных
- `internal/service` - бизнес-логика
- `internal/storage` - хранилища данных

## Примеры использования

### Пример 1: Переиспользование структур

```go
// generate:reset
type Request struct {
    ID   int
    Data []byte
}

var pool = sync.Pool{
    New: func() interface{} {
        return &Request{
            Data: make([]byte, 0, 1024),
        }
    },
}

func GetRequest() *Request {
    return pool.Get().(*Request)
}

func PutRequest(r *Request) {
    r.Reset() // Очищаем перед возвратом в пул
    pool.Put(r)
}
```

### Пример 2: Graceful shutdown

```go
func main() {
    os.Exit(run())
}

func run() int {
    defer cleanup() // Этот defer выполнится
    
    if err := startServer(); err != nil {
        log.Printf("Error: %v", err)
        return 1
    }
    return 0
}
```

## Требования

- Go 1.21+ (для функции `clear()`)
- Зависимости устанавливаются автоматически через `go mod`

## Полезные команды

```bash
# Обновление зависимостей
go get -u ./...
go mod tidy

# Проверка на уязвимости
go list -json -m all | nancy sleuth

# Профилирование
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.
go tool pprof cpu.prof

# Бенчмарки
go test -bench=. -benchmem ./...
```

## Troubleshooting

### Проблема: Генератор не находит структуры

**Решение:**
1. Проверьте комментарий `// generate:reset` (должен быть непосредственно над `type`)
2. Запускайте генератор из корня проекта
3. Убедитесь, что файл не `_test.go` и не `.gen.go`

### Проблема: Staticlint находит много ошибок

**Решение:**
1. Обработайте все возвращаемые ошибки
2. Не используйте `os.Exit()` напрямую в `main()`
3. Закрывайте HTTP response body: `defer resp.Body.Close()`

### Проблема: Тесты падают после генерации

**Решение:**
1. Перезапустите генератор: `go run cmd/reset/main.go`
2. Проверьте, что все поля структуры экспортированы (с большой буквы)
3. Убедитесь, что используется Go 1.21+ для функции `clear()`

## Контрибьюция

При добавлении нового кода:

1. Добавьте godoc документацию
2. Напишите тесты
3. Запустите `make check` перед коммитом
4. Используйте `// generate:reset` для структур, требующих сброса
5. Следуйте принципам dependency injection

## Ссылки

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go AST Documentation](https://pkg.go.dev/go/ast)
- [Staticcheck Documentation](https://staticcheck.io/docs/)
