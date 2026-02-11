// Package main реализует multichecker - инструмент статического анализа кода Go.
//
// # Описание
//
// Multichecker объединяет множество анализаторов из различных источников:
//   - Стандартные анализаторы из golang.org/x/tools/go/analysis/passes
//   - Анализаторы staticcheck.io (все SA классы и дополнительные)
//   - Публичные анализаторы сторонних разработчиков
//   - Собственный анализатор osexit
//
// # Запуск
//
// Для запуска multichecker используйте команду:
//
//	go run cmd/staticlint/main.go ./...
//
// или после сборки:
//
//	go build -o staticlint cmd/staticlint/main.go
//	./staticlint ./...
//
// # Включенные анализаторы
//
// ## Стандартные анализаторы (golang.org/x/tools/go/analysis/passes):
//
//   - asmdecl: проверяет соответствие между Go объявлениями и assembly файлами
//   - assign: обнаруживает бесполезные присваивания
//   - atomic: проверяет распространенные ошибки использования пакета sync/atomic
//   - bools: обнаруживает распространенные ошибки с булевыми операторами
//   - buildtag: проверяет корректность build tags
//   - cgocall: обнаруживает нарушения правил передачи указателей в cgo
//   - errorsas: проверяет, что второй аргумент errors.As является указателем
//   - httpresponse: проверяет ошибки использования HTTP responses
//   - ifaceassert: обнаруживает невозможные утверждения интерфейсов
//   - loopclosure: проверяет ссылки на переменные цикла из вложенных функций
//   - lostcancel: проверяет, что cancel функции context вызываются
//   - nilfunc: проверяет бесполезные сравнения функций с nil
//   - printf: проверяет согласованность строк формата Printf и аргументов
//   - shift: проверяет сдвиги, которые превышают ширину целого числа
//   - stdmethods: проверяет сигнатуры известных методов интерфейсов
//   - structtag: проверяет правильность тегов структур
//   - tests: проверяет распространенные ошибочные использования тестов и примеров
//   - unmarshal: проверяет передачу не-указателей или не-интерфейсов в unmarshal
//   - unreachable: проверяет недостижимый код
//   - unsafeptr: проверяет недопустимые преобразования uintptr в unsafe.Pointer
//   - unusedresult: проверяет неиспользуемые результаты вызовов определенных функций
//
// ## Анализаторы staticcheck.io:
//
// ### Класс SA (Static Analysis):
//   - SA1000-SA1030: различные проверки на ошибки в коде
//   - SA2000-SA2003: проверки на конкурентность и синхронизацию
//   - SA3000-SA3001: проверки на тестирование
//   - SA4000-SA4031: проверки на правильность использования стандартной библиотеки
//   - SA5000-SA5012: проверки на корректность
//   - SA6000-SA6005: проверки на производительность
//   - SA9000-SA9008: проверки на подозрительный код
//
// ### Класс S (Simple):
//   - S1000: проверяет упрощаемые выражения
//
// ### Класс ST (Style):
//   - ST1000: проверяет стиль кода
//
// ### Класс QF (Quick Fix):
//   - QF1001: проверяет возможности упрощения кода
//
// ## Публичные анализаторы:
//
//   - errcheck (github.com/kisielk/errcheck): проверяет, что ошибки обрабатываются
//   - bodyclose (github.com/timakin/bodyclose): проверяет, что HTTP response body закрывается
//
// ## Собственный анализатор:
//
//   - osexit: запрещает прямой вызов os.Exit в функции main пакета main.
//     Это помогает обеспечить корректное завершение программы с выполнением
//     всех defer'ов и правильной очисткой ресурсов.
//
// # Примеры использования
//
// Проверка текущего пакета:
//
//	staticlint .
//
// Проверка всех пакетов проекта:
//
//	staticlint ./...
//
// Проверка конкретного пакета:
//
//	staticlint ./internal/handler
//
// # Конфигурация
//
// Multichecker не требует дополнительной конфигурации и работает "из коробки".
// Все анализаторы включены по умолчанию.
package main

import (
	"strings"

	"awesome-shortener/cmd/staticlint/osexit"
	"github.com/kisielk/errcheck/errcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	// Собираем все анализаторы
	var analyzers []*analysis.Analyzer

	// Добавляем стандартные анализаторы из golang.org/x/tools/go/analysis/passes
	analyzers = append(analyzers,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		errorsas.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		stdmethods.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
	)

	// Добавляем анализаторы из staticcheck.io
	for _, v := range staticcheck.Analyzers {
		// Добавляем все анализаторы класса SA
		if strings.HasPrefix(v.Analyzer.Name, "SA") {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	// Добавляем дополнительные анализаторы из staticcheck (не SA класса)
	for _, v := range staticcheck.Analyzers {
		name := v.Analyzer.Name
		// Добавляем по одному анализатору из других классов
		if strings.HasPrefix(name, "S1000") || // Simple
			strings.HasPrefix(name, "ST1000") || // Style
			strings.HasPrefix(name, "QF1001") { // Quick Fix
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	// Добавляем публичные анализаторы
	analyzers = append(analyzers,
		errcheck.Analyzer,  // проверка обработки ошибок
		bodyclose.Analyzer, // проверка закрытия HTTP response body
	)

	// Добавляем собственный анализатор
	analyzers = append(analyzers, osexit.Analyzer)

	// Запускаем multichecker
	multichecker.Main(analyzers...)
}
