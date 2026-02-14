# Staticlint - Multichecker для статического анализа кода

## Описание

Staticlint - это инструмент статического анализа кода Go, объединяющий множество анализаторов из различных источников для комплексной проверки качества кода.

## Состав анализаторов

### 1. Стандартные анализаторы (golang.org/x/tools/go/analysis/passes)

Включены следующие анализаторы:

- **asmdecl** - проверяет соответствие между Go объявлениями и assembly файлами
- **assign** - обнаруживает бесполезные присваивания
- **atomic** - проверяет распространенные ошибки использования пакета sync/atomic
- **bools** - обнаруживает распространенные ошибки с булевыми операторами
- **buildtag** - проверяет корректность build tags
- **cgocall** - обнаруживает нарушения правил передачи указателей в cgo
- **errorsas** - проверяет, что второй аргумент errors.As является указателем
- **httpresponse** - проверяет ошибки использования HTTP responses
- **ifaceassert** - обнаруживает невозможные утверждения интерфейсов
- **loopclosure** - проверяет ссылки на переменные цикла из вложенных функций
- **lostcancel** - проверяет, что cancel функции context вызываются
- **nilfunc** - проверяет бесполезные сравнения функций с nil
- **printf** - проверяет согласованность строк формата Printf и аргументов
- **shift** - проверяет сдвиги, которые превышают ширину целого числа
- **stdmethods** - проверяет сигнатуры известных методов интерфейсов
- **structtag** - проверяет правильность тегов структур
- **tests** - проверяет распространенные ошибочные использования тестов и примеров
- **unmarshal** - проверяет передачу не-указателей или не-интерфейсов в unmarshal
- **unreachable** - проверяет недостижимый код
- **unsafeptr** - проверяет недопустимые преобразования uintptr в unsafe.Pointer
- **unusedresult** - проверяет неиспользуемые результаты вызовов определенных функций

### 2. Анализаторы staticcheck.io

#### Класс SA (Static Analysis) - все анализаторы

Включены все анализаторы класса SA, которые обнаруживают различные ошибки в коде:
- SA1xxx - различные проверки на ошибки
- SA2xxx - проверки на конкурентность и синхронизацию
- SA3xxx - проверки на тестирование
- SA4xxx - проверки на правильность использования стандартной библиотеки
- SA5xxx - проверки на корректность
- SA6xxx - проверки на производительность
- SA9xxx - проверки на подозрительный код

#### Дополнительные классы

- **S1000** (Simple) - проверяет упрощаемые выражения
- **ST1000** (Style) - проверяет стиль кода
- **QF1001** (Quick Fix) - проверяет возможности упрощения кода

### 3. Публичные анализаторы

- **errcheck** (github.com/kisielk/errcheck) - проверяет, что все возвращаемые ошибки обрабатываются
- **bodyclose** (github.com/timakin/bodyclose) - проверяет, что HTTP response body корректно закрывается

### 4. Собственный анализатор

- **osexit** - запрещает прямой вызов os.Exit в функции main пакета main

## Собственный анализатор osexit

### Назначение

Анализатор `osexit` запрещает использование прямого вызова `os.Exit()` в функции `main` пакета `main`. Это важно для обеспечения корректного завершения программы с выполнением всех отложенных операций (defer) и правильной очисткой ресурсов.

### Проблема

Когда `os.Exit()` вызывается напрямую в `main`, программа завершается немедленно, минуя выполнение всех `defer` операций. Это может привести к:
- Незакрытым файлам и соединениям
- Неотправленным логам
- Неосвобожденным ресурсам
- Некорректному состоянию данных

### Решение

Вместо прямого вызова `os.Exit()` в `main`, рекомендуется:

**Неправильно:**
```go
package main

import "os"

func main() {
    if err := doSomething(); err != nil {
        os.Exit(1) // Плохо: defer'ы не выполнятся
    }
}
```

**Правильно:**
```go
package main

import "os"

func main() {
    os.Exit(run())
}

func run() int {
    defer cleanup() // Этот defer выполнится
    
    if err := doSomething(); err != nil {
        return 1
    }
    return 0
}
```

## Установка

### Из исходников

```bash
cd cmd/staticlint
go build -o staticlint
```

### Установка в GOPATH

```bash
go install ./cmd/staticlint
```

## Использование

### Базовое использование

Проверка текущего пакета:
```bash
staticlint .
```

Проверка всех пакетов проекта:
```bash
staticlint ./...
```

Проверка конкретного пакета:
```bash
staticlint ./internal/handler
```

### Запуск через go run

```bash
go run ./cmd/staticlint/main.go ./...
```

### Интеграция в CI/CD

Добавьте в ваш CI/CD pipeline:

```yaml
# GitHub Actions
- name: Run staticlint
  run: |
    go build -o staticlint ./cmd/staticlint
    ./staticlint ./...
```

```yaml
# GitLab CI
lint:
  script:
    - go build -o staticlint ./cmd/staticlint
    - ./staticlint ./...
```

## Примеры обнаруженных проблем

### Пример 1: Необработанная ошибка (errcheck)

```go
// Плохо
file, _ := os.Open("file.txt")

// Хорошо
file, err := os.Open("file.txt")
if err != nil {
    return err
}
```

### Пример 2: Незакрытый response body (bodyclose)

```go
// Плохо
resp, err := http.Get(url)
if err != nil {
    return err
}
// body не закрыт!

// Хорошо
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()
```

### Пример 3: os.Exit в main (osexit)

```go
// Плохо
func main() {
    os.Exit(1) // defer'ы не выполнятся
}

// Хорошо
func main() {
    os.Exit(run())
}

func run() int {
    defer cleanup()
    return 0
}
```

## Конфигурация

Staticlint не требует дополнительной конфигурации и работает "из коробки". Все анализаторы включены по умолчанию.

## Разработка

### Добавление нового анализатора

1. Импортируйте анализатор в `main.go`
2. Добавьте его в срез `analyzers`
3. Обновите документацию

```go
import "path/to/analyzer"

func main() {
    analyzers = append(analyzers, analyzer.Analyzer)
    multichecker.Main(analyzers...)
}
```

### Тестирование собственного анализатора

```bash
cd cmd/staticlint/osexit
go test -v
```

## Зависимости

- golang.org/x/tools/go/analysis
- honnef.co/go/tools/staticcheck
- github.com/kisielk/errcheck
- github.com/timakin/bodyclose

## Лицензия

Этот инструмент является частью проекта awesome-shortener.
