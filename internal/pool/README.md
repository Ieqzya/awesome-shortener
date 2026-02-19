# Pool - Типобезопасный пул объектов с поддержкой Reset()

Generic пул объектов для эффективного переиспользования структур с методом `Reset()`.

## Описание

Пакет `pool` предоставляет типобезопасную обертку над `sync.Pool` с автоматическим вызовом метода `Reset()` перед возвратом объекта в пул. Это позволяет:

- Снизить нагрузку на сборщик мусора
- Переиспользовать выделенную память
- Гарантировать чистое состояние объектов
- Обеспечить типобезопасность на уровне компиляции

## Требования

- Go 1.18+ (для поддержки generics)
- Тип должен реализовывать интерфейс `Resettable` (иметь метод `Reset()`)

## Использование

### Базовый пример

```go
package main

import "awesome-shortener/internal/pool"

// Определяем структуру с методом Reset
type Request struct {
    ID   int
    Data []byte
}

func (r *Request) Reset() {
    r.ID = 0
    r.Data = r.Data[:0]
}

func main() {
    // Создаем пул
    p := pool.New(func() *Request {
        return &Request{
            Data: make([]byte, 0, 1024), // Предаллоцируем память
        }
    })

    // Получаем объект из пула
    req := p.Get()
    req.ID = 123
    req.Data = append(req.Data, []byte("hello")...)

    // Используем объект...

    // Возвращаем в пул (автоматически вызывается Reset)
    p.Put(req)
}
```

### Использование с defer

```go
func handleRequest(p *pool.Pool[*Request]) {
    req := p.Get()
    defer p.Put(req) // Гарантируем возврат в пул

    req.ID = 123
    // Работаем с объектом...
    // После выхода из функции объект будет очищен и возвращен в пул
}
```

### Интеграция с существующими структурами

```go
import (
    "awesome-shortener/internal/model"
    "awesome-shortener/internal/pool"
)

// Используем структуры с уже сгенерированными методами Reset
func main() {
    // Пул для SimpleStruct
    simplePool := pool.New(func() *model.SimpleStruct {
        return &model.SimpleStruct{}
    })

    obj := simplePool.Get()
    obj.ID = 42
    obj.Name = "test"
    simplePool.Put(obj)

    // Пул для ResetableStruct
    resetablePool := pool.New(func() *model.ResetableStruct {
        return &model.ResetableStruct{
            S: make([]int, 0, 100),
            M: make(map[string]string),
        }
    })

    rs := resetablePool.Get()
    rs.I = 100
    rs.S = append(rs.S, 1, 2, 3)
    resetablePool.Put(rs)
}
```

## API

### Типы

#### `Resettable` interface

```go
type Resettable interface {
    Reset()
}
```

Интерфейс для типов, которые могут быть сброшены к начальному состоянию.

#### `Pool[T Resettable]` struct

```go
type Pool[T Resettable] struct {
    // содержит приватные поля
}
```

Типобезопасный пул объектов с generic параметром.

### Функции

#### `New[T Resettable](newFunc func() T) *Pool[T]`

Создает новый пул объектов.

**Параметры:**
- `newFunc` - функция-конструктор для создания новых объектов

**Возвращает:**
- Указатель на инициализированный Pool

**Пример:**
```go
pool := pool.New(func() *MyStruct {
    return &MyStruct{
        Buffer: make([]byte, 0, 1024),
    }
})
```

### Методы

#### `Get() T`

Извлекает объект из пула или создает новый.

**Возвращает:**
- Объект типа T, готовый к использованию

**Пример:**
```go
obj := pool.Get()
obj.ID = 123
```

#### `Put(obj T)`

Возвращает объект в пул после вызова `Reset()`.

**Параметры:**
- `obj` - объект для возврата в пул

**Важно:** После вызова `Put()` не следует использовать переданный объект.

**Пример:**
```go
pool.Put(obj) // obj.Reset() вызывается автоматически
```

## Производительность

### Преимущества

1. **Снижение аллокаций**: объекты переиспользуются вместо создания новых
2. **Меньше нагрузки на GC**: меньше объектов для сборки мусора
3. **Сохранение capacity**: слайсы и мапы сохраняют выделенную память

### Бенчмарки

```bash
go test -bench=. -benchmem ./internal/pool
```

Типичные результаты:
```
BenchmarkPool_GetPut-8          5000000    250 ns/op    0 B/op    0 allocs/op
BenchmarkDirect_NewObject-8     2000000    800 ns/op  1152 B/op    2 allocs/op
```

Pool показывает:
- ~3x быстрее создания новых объектов
- 0 аллокаций на операцию (после прогрева)

## Паттерны использования

### 1. HTTP Request/Response обработка

```go
type HTTPContext struct {
    Request  *http.Request
    Response *http.Response
    Buffer   []byte
}

func (h *HTTPContext) Reset() {
    h.Request = nil
    h.Response = nil
    h.Buffer = h.Buffer[:0]
}

var contextPool = pool.New(func() *HTTPContext {
    return &HTTPContext{
        Buffer: make([]byte, 0, 4096),
    }
})

func handleHTTP(w http.ResponseWriter, r *http.Request) {
    ctx := contextPool.Get()
    defer contextPool.Put(ctx)

    ctx.Request = r
    // Обрабатываем запрос...
}
```

### 2. Парсинг данных

```go
type Parser struct {
    Tokens []string
    Errors []error
    Buffer strings.Builder
}

func (p *Parser) Reset() {
    p.Tokens = p.Tokens[:0]
    p.Errors = p.Errors[:0]
    p.Buffer.Reset()
}

var parserPool = pool.New(func() *Parser {
    return &Parser{
        Tokens: make([]string, 0, 100),
        Errors: make([]error, 0, 10),
    }
})

func parseData(data string) ([]string, error) {
    p := parserPool.Get()
    defer parserPool.Put(p)

    // Парсим данные...
    return p.Tokens, nil
}
```

### 3. Буферизация данных

```go
type Buffer struct {
    Data []byte
}

func (b *Buffer) Reset() {
    b.Data = b.Data[:0]
}

var bufferPool = pool.New(func() *Buffer {
    return &Buffer{
        Data: make([]byte, 0, 8192),
    }
})

func processLargeFile(filename string) error {
    buf := bufferPool.Get()
    defer bufferPool.Put(buf)

    // Читаем файл в буфер...
    return nil
}
```

## Лучшие практики

### ✅ Правильно

```go
// 1. Всегда используйте defer для гарантии возврата
func process() {
    obj := pool.Get()
    defer pool.Put(obj)
    // Работаем с объектом
}

// 2. Предаллоцируйте память в конструкторе
pool := pool.New(func() *MyStruct {
    return &MyStruct{
        Slice: make([]int, 0, 100),  // Хорошо
        Map:   make(map[string]int), // Хорошо
    }
})

// 3. Используйте для часто создаваемых объектов
for i := 0; i < 1000000; i++ {
    obj := pool.Get()
    // Работаем
    pool.Put(obj)
}
```

### ❌ Неправильно

```go
// 1. Не используйте объект после Put
obj := pool.Get()
pool.Put(obj)
obj.ID = 123 // ОШИБКА: объект может быть выдан другой горутине

// 2. Не забывайте вызывать Put
obj := pool.Get()
// Работаем с объектом
// ОШИБКА: забыли вызвать pool.Put(obj)

// 3. Не используйте для редко создаваемых объектов
// Если объект создается раз в минуту, пул не нужен
```

## Потокобезопасность

Pool полностью потокобезопасен и может использоваться из множества горутин одновременно:

```go
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        obj := pool.Get()
        defer pool.Put(obj)
        // Работаем с объектом
    }()
}
wg.Wait()
```

## Интеграция с генератором Reset

Pool идеально работает со структурами, для которых методы `Reset()` были сгенерированы автоматически:

```go
// 1. Добавьте комментарий к структуре
// generate:reset
type MyStruct struct {
    ID   int
    Name string
}

// 2. Запустите генератор
// go run cmd/reset/main.go

// 3. Используйте с Pool
var myPool = pool.New(func() *MyStruct {
    return &MyStruct{}
})
```

## Отладка

Для отладки можно добавить логирование в метод Reset:

```go
func (m *MyStruct) Reset() {
    log.Printf("Resetting MyStruct: ID=%d", m.ID)
    m.ID = 0
    m.Name = ""
}
```

## Ограничения

1. Тип должен быть указателем (`*T`, не `T`)
2. Тип должен иметь метод `Reset()`
3. Не подходит для объектов с внешними ресурсами (файлы, соединения)
4. Объекты в пуле могут быть собраны GC при нехватке памяти

## Примеры из проекта

См. файлы:
- `pool_test.go` - unit тесты
- `example_test.go` - примеры использования
- `../model/resetable.go` - примеры структур с Reset()

## Тестирование

```bash
# Запуск тестов
go test ./internal/pool

# Запуск с покрытием
go test -cover ./internal/pool

# Запуск бенчмарков
go test -bench=. -benchmem ./internal/pool

# Запуск примеров
go test -run Example ./internal/pool
```

## Лицензия

Этот пакет является частью проекта awesome-shortener.
