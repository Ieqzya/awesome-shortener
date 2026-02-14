# Reset Generator

Генератор методов `Reset()` для структур Go с автоматическим сбросом полей к начальным значениям.

## Описание

Reset Generator сканирует все пакеты проекта, находит структуры с комментарием `// generate:reset` и автоматически генерирует для них методы `Reset()`, которые сбрасывают состояние объекта к начальным значениям.

## Правила генерации

Метод `Reset()` сбрасывает поля структуры по следующим правилам:

1. **Примитивные типы** приводятся к нулевым значениям:
   - `int`, `int8`, `int16`, `int32`, `int64` → `0`
   - `uint`, `uint8`, `uint16`, `uint32`, `uint64` → `0`
   - `float32`, `float64` → `0`
   - `string` → `""`
   - `bool` → `false`

2. **Слайсы** обрезаются по длине, но не зануляются:
   - `[]int{1, 2, 3}` → `[]int{}` (используется `slice[:0]`)
   - Сохраняется выделенная память (capacity)

3. **Мапы** очищаются с помощью встроенной функции `clear()`:
   - `map[string]int{"a": 1}` → `map[string]int{}`

4. **Вложенные структуры** с методом `Reset()` вызывают этот метод:
   - Если структура имеет метод `Reset()`, он будет вызван
   - Иначе структура сбрасывается к нулевому значению

5. **Указатели** (не-nil) сбрасывают свои значения:
   - Проверяется на `nil`
   - Если не `nil`, значение сбрасывается по правилам выше

## Использование

### Шаг 1: Добавьте комментарий

Добавьте комментарий `// generate:reset` над структурой:

```go
package mypackage

// generate:reset
type User struct {
    ID       int
    Name     string
    Email    string
    Tags     []string
    Metadata map[string]string
}
```

### Шаг 2: Запустите генератор

```bash
go run cmd/reset/main.go
```

### Шаг 3: Используйте сгенерированный метод

```go
user := &User{
    ID:       123,
    Name:     "John Doe",
    Email:    "john@example.com",
    Tags:     []string{"admin", "user"},
    Metadata: map[string]string{"role": "admin"},
}

// Сбрасываем все поля
user.Reset()

// Теперь:
// user.ID == 0
// user.Name == ""
// user.Email == ""
// len(user.Tags) == 0 (но capacity сохранен)
// len(user.Metadata) == 0
```

## Примеры

### Пример 1: Простая структура

**Входной код:**

```go
// generate:reset
type Config struct {
    Host string
    Port int
    SSL  bool
}
```

**Сгенерированный код:**

```go
// Reset сбрасывает все поля Config к начальным значениям
func (c *Config) Reset() {
    if c == nil {
        return
    }

    c.Host = ""
    c.Port = 0
    c.SSL = false
}
```

### Пример 2: Структура со слайсами и мапами

**Входной код:**

```go
// generate:reset
type Cache struct {
    Items []string
    Index map[string]int
}
```

**Сгенерированный код:**

```go
// Reset сбрасывает все поля Cache к начальным значениям
func (c *Cache) Reset() {
    if c == nil {
        return
    }

    c.Items = c.Items[:0]
    clear(c.Index)
}
```

### Пример 3: Структура с указателями

**Входной код:**

```go
// generate:reset
type Request struct {
    ID      int
    UserID  *int
    Message *string
}
```

**Сгенерированный код:**

```go
// Reset сбрасывает все поля Request к начальным значениям
func (r *Request) Reset() {
    if r == nil {
        return
    }

    r.ID = 0
    if r.UserID != nil {
        *r.UserID = 0
    }
    if r.Message != nil {
        *r.Message = ""
    }
}
```

### Пример 4: Вложенные структуры

**Входной код:**

```go
// generate:reset
type Address struct {
    Street string
    City   string
}

// generate:reset
type Person struct {
    Name    string
    Age     int
    Address Address
    HomePtr *Address
}
```

**Сгенерированный код:**

```go
// Reset сбрасывает все поля Address к начальным значениям
func (a *Address) Reset() {
    if a == nil {
        return
    }

    a.Street = ""
    a.City = ""
}

// Reset сбрасывает все поля Person к начальным значениям
func (p *Person) Reset() {
    if p == nil {
        return
    }

    p.Name = ""
    p.Age = 0
    if resetter, ok := interface{}(p.Address).(interface{ Reset() }); ok {
        resetter.Reset()
    } else {
        p.Address = Address{}
    }
    if p.HomePtr != nil {
        if resetter, ok := interface{}(p.HomePtr).(interface{ Reset() }); ok {
            resetter.Reset()
        } else {
            *p.HomePtr = Address{}
        }
    }
}
```

## Интеграция в проект

### Добавление в Makefile

```makefile
.PHONY: generate
generate:
	go run cmd/reset/main.go
```

### Добавление в go:generate

Добавьте в любой файл проекта:

```go
//go:generate go run cmd/reset/main.go
```

Затем запустите:

```bash
go generate ./...
```

### Интеграция в CI/CD

```yaml
# GitHub Actions
- name: Generate Reset methods
  run: go run cmd/reset/main.go

- name: Check for changes
  run: |
    if [[ -n $(git status --porcelain) ]]; then
      echo "Generated files are not up to date"
      exit 1
    fi
```

## Особенности реализации

### Производительность

- **Слайсы**: используется `slice[:0]` вместо `nil`, что сохраняет выделенную память и избегает повторных аллокаций
- **Мапы**: используется встроенная функция `clear()` (Go 1.21+), которая эффективно очищает мапу без переаллокации

### Безопасность

- Все методы проверяют receiver на `nil`
- Указатели проверяются на `nil` перед разыменованием
- Вложенные структуры проверяются на наличие метода `Reset()` через type assertion

### Ограничения

- Генератор не обрабатывает приватные поля (с маленькой буквы)
- Не поддерживаются каналы и функции
- Интерфейсы сбрасываются к нулевому значению
- Требуется Go 1.21+ для функции `clear()`

## Структура сгенерированных файлов

Для каждого пакета создается файл `reset.gen.go`:

```
project/
├── internal/
│   ├── model/
│   │   ├── user.go
│   │   └── reset.gen.go    # Сгенерированные методы для model
│   └── cache/
│       ├── cache.go
│       └── reset.gen.go    # Сгенерированные методы для cache
└── cmd/
    └── reset/
        └── main.go         # Генератор
```

## Отладка

Если генератор не находит структуры:

1. Проверьте, что комментарий `// generate:reset` находится непосредственно над объявлением типа
2. Убедитесь, что файл не является тестовым (`_test.go`)
3. Проверьте, что файл не является сгенерированным (`.gen.go`)

Если генерация завершается с ошибкой:

1. Проверьте синтаксис Go файлов: `go vet ./...`
2. Убедитесь, что все импорты корректны
3. Проверьте логи генератора для деталей ошибки

## Примеры использования в проекте

### Пул объектов

```go
// generate:reset
type Request struct {
    ID   int
    Data []byte
}

var requestPool = sync.Pool{
    New: func() interface{} {
        return &Request{
            Data: make([]byte, 0, 1024),
        }
    },
}

func GetRequest() *Request {
    return requestPool.Get().(*Request)
}

func PutRequest(r *Request) {
    r.Reset() // Очищаем перед возвратом в пул
    requestPool.Put(r)
}
```

### Переиспользование структур

```go
// generate:reset
type QueryBuilder struct {
    query  strings.Builder
    params []interface{}
    errors []error
}

func (qb *QueryBuilder) Build() (string, []interface{}, error) {
    defer qb.Reset() // Автоматическая очистка после использования
    
    // ... логика построения запроса
    
    return qb.query.String(), qb.params, nil
}
```

## Лицензия

Этот инструмент является частью проекта awesome-shortener.
