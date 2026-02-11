// Package main реализует генератор методов Reset() для структур с комментарием // generate:reset.
//
// # Описание
//
// Генератор сканирует все пакеты проекта, начиная с корневой директории,
// находит структуры с комментарием // generate:reset и генерирует для них
// методы Reset(), которые сбрасывают состояние объекта к начальным значениям.
//
// # Правила генерации
//
// Метод Reset() сбрасывает поля структуры по следующим правилам:
//   - Примитивы приводятся к нулевым значениям (int -> 0, string -> "", bool -> false)
//   - Слайсы обрезаются по длине (slice[:0]), но не зануляются
//   - Мапы очищаются с помощью встроенной функции clear()
//   - Вложенные структуры с методом Reset() вызывают этот метод
//   - Не-nil указатели сбрасывают свои значения по правилам выше
//
// # Использование
//
// Добавьте комментарий // generate:reset над структурой:
//
//	// generate:reset
//	type MyStruct struct {
//		ID   int
//		Name string
//	}
//
// Запустите генератор:
//
//	go run cmd/reset/main.go
//
// Будет создан файл reset.gen.go с методом Reset() для MyStruct.
//
// # Пример сгенерированного кода
//
//	func (m *MyStruct) Reset() {
//		if m == nil {
//			return
//		}
//		m.ID = 0
//		m.Name = ""
//	}
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// StructInfo содержит информацию о структуре для генерации
type StructInfo struct {
	Name       string      // имя структуры
	RecvName   string      // имя receiver'а (первая буква имени структуры в нижнем регистре)
	Fields     []FieldInfo // поля структуры
	PackageName string     // имя пакета
}

// FieldInfo содержит информацию о поле структуры
type FieldInfo struct {
	Name     string // имя поля
	Type     string // тип поля
	IsSlice  bool   // является ли слайсом
	IsMap    bool   // является ли мапой
	IsPtr    bool   // является ли указателем
	ElemType string // тип элемента (для указателей, слайсов)
}

func main() {
	// Получаем текущую директорию
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Ошибка получения текущей директории: %v", err)
	}

	// Сканируем все пакеты
	packages, err := scanPackages(rootDir)
	if err != nil {
		log.Fatalf("Ошибка сканирования пакетов: %v", err)
	}

	// Генерируем методы Reset для каждого пакета
	for pkgPath, structs := range packages {
		if len(structs) == 0 {
			continue
		}

		if err := generateResetFile(pkgPath, structs); err != nil {
			log.Printf("Ошибка генерации для пакета %s: %v", pkgPath, err)
		} else {
			fmt.Printf("Сгенерирован файл reset.gen.go для пакета %s (%d структур)\n", pkgPath, len(structs))
		}
	}
}

// scanPackages сканирует все пакеты начиная с rootDir
func scanPackages(rootDir string) (map[string][]StructInfo, error) {
	packages := make(map[string][]StructInfo)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем директории vendor, .git и т.д.
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Обрабатываем только .go файлы (кроме _test.go и .gen.go)
		if !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") ||
			strings.HasSuffix(path, ".gen.go") {
			return nil
		}

		// Парсим файл
		structs, err := parseFile(path)
		if err != nil {
			log.Printf("Предупреждение: ошибка парсинга %s: %v", path, err)
			return nil
		}

		if len(structs) > 0 {
			pkgDir := filepath.Dir(path)
			packages[pkgDir] = append(packages[pkgDir], structs...)
		}

		return nil
	})

	return packages, err
}

// parseFile парсит файл и возвращает структуры с комментарием // generate:reset
func parseFile(filename string) ([]StructInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var structs []StructInfo

	// Проходим по всем объявлениям
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		// Проверяем комментарий
		if !hasGenerateResetComment(genDecl.Doc) {
			continue
		}

		// Обрабатываем каждую спецификацию типа
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Извлекаем информацию о структуре
			structInfo := StructInfo{
				Name:        typeSpec.Name.Name,
				RecvName:    strings.ToLower(typeSpec.Name.Name[:1]),
				PackageName: node.Name.Name,
			}

			// Извлекаем информацию о полях
			for _, field := range structType.Fields.List {
				for _, name := range field.Names {
					fieldInfo := analyzeField(name.Name, field.Type)
					structInfo.Fields = append(structInfo.Fields, fieldInfo)
				}
			}

			structs = append(structs, structInfo)
		}
	}

	return structs, nil
}

// hasGenerateResetComment проверяет наличие комментария // generate:reset
func hasGenerateResetComment(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}

	for _, comment := range doc.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if strings.HasPrefix(text, "generate:reset") {
			return true
		}
	}

	return false
}

// analyzeField анализирует тип поля
func analyzeField(name string, fieldType ast.Expr) FieldInfo {
	info := FieldInfo{Name: name}

	switch t := fieldType.(type) {
	case *ast.Ident:
		// Простой тип (int, string, bool, etc.)
		info.Type = t.Name

	case *ast.StarExpr:
		// Указатель
		info.IsPtr = true
		info.ElemType = exprToString(t.X)
		info.Type = "*" + info.ElemType

	case *ast.ArrayType:
		// Слайс или массив
		if t.Len == nil {
			// Слайс
			info.IsSlice = true
			info.ElemType = exprToString(t.Elt)
			info.Type = "[]" + info.ElemType
		} else {
			// Массив
			info.Type = exprToString(fieldType)
		}

	case *ast.MapType:
		// Мапа
		info.IsMap = true
		info.Type = exprToString(fieldType)

	default:
		info.Type = exprToString(fieldType)
	}

	return info
}

// exprToString преобразует ast.Expr в строку
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprToString(t.Elt)
		}
		return "[" + exprToString(t.Len) + "]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// generateResetFile генерирует файл reset.gen.go для пакета
func generateResetFile(pkgPath string, structs []StructInfo) error {
	var buf bytes.Buffer

	// Получаем имя пакета из первой структуры
	packageName := structs[0].PackageName

	// Генерируем заголовок файла
	buf.WriteString("// Code generated by cmd/reset. DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// Генерируем методы Reset для каждой структуры
	for _, structInfo := range structs {
		generateResetMethod(&buf, structInfo)
		buf.WriteString("\n")
	}

	// Форматируем код
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("ошибка форматирования кода: %w\n%s", err, buf.String())
	}

	// Записываем в файл
	outputPath := filepath.Join(pkgPath, "reset.gen.go")
	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	return nil
}

// generateResetMethod генерирует метод Reset для структуры
func generateResetMethod(buf *bytes.Buffer, info StructInfo) {
	// Генерируем сигнатуру метода
	buf.WriteString(fmt.Sprintf("// Reset сбрасывает все поля %s к начальным значениям\n", info.Name))
	buf.WriteString(fmt.Sprintf("func (%s *%s) Reset() {\n", info.RecvName, info.Name))

	// Проверка на nil
	buf.WriteString(fmt.Sprintf("\tif %s == nil {\n", info.RecvName))
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n\n")

	// Генерируем код сброса для каждого поля
	for _, field := range info.Fields {
		generateFieldReset(buf, info.RecvName, field)
	}

	buf.WriteString("}\n")
}

// generateFieldReset генерирует код сброса для поля
func generateFieldReset(buf *bytes.Buffer, recvName string, field FieldInfo) {
	fieldAccess := fmt.Sprintf("%s.%s", recvName, field.Name)

	if field.IsSlice {
		// Слайс: обрезаем до нулевой длины
		buf.WriteString(fmt.Sprintf("\t%s = %s[:0]\n", fieldAccess, fieldAccess))
		return
	}

	if field.IsMap {
		// Мапа: очищаем
		buf.WriteString(fmt.Sprintf("\tclear(%s)\n", fieldAccess))
		return
	}

	if field.IsPtr {
		// Указатель: проверяем на nil и сбрасываем значение
		buf.WriteString(fmt.Sprintf("\tif %s != nil {\n", fieldAccess))

		// Проверяем, является ли тип структурой с методом Reset
		if isCustomType(field.ElemType) {
			buf.WriteString(fmt.Sprintf("\t\tif resetter, ok := interface{}(%s).(interface{ Reset() }); ok {\n", fieldAccess))
			buf.WriteString("\t\t\tresetter.Reset()\n")
			buf.WriteString("\t\t} else {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t*%s = %s\n", fieldAccess, getZeroValue(field.ElemType)))
			buf.WriteString("\t\t}\n")
		} else {
			buf.WriteString(fmt.Sprintf("\t\t*%s = %s\n", fieldAccess, getZeroValue(field.ElemType)))
		}

		buf.WriteString("\t}\n")
		return
	}

	// Проверяем, является ли поле структурой с методом Reset
	if isCustomType(field.Type) {
		buf.WriteString(fmt.Sprintf("\tif resetter, ok := interface{}(%s).(interface{ Reset() }); ok {\n", fieldAccess))
		buf.WriteString("\t\tresetter.Reset()\n")
		buf.WriteString("\t} else {\n")
		buf.WriteString(fmt.Sprintf("\t\t%s = %s\n", fieldAccess, getZeroValue(field.Type)))
		buf.WriteString("\t}\n")
		return
	}

	// Обычное поле: присваиваем нулевое значение
	buf.WriteString(fmt.Sprintf("\t%s = %s\n", fieldAccess, getZeroValue(field.Type)))
}

// getZeroValue возвращает нулевое значение для типа
func getZeroValue(typeName string) string {
	switch typeName {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"byte", "rune", "float32", "float64", "complex64", "complex128":
		return "0"
	case "string":
		return `""`
	case "bool":
		return "false"
	default:
		// Для пользовательских типов возвращаем нулевое значение типа
		return typeName + "{}"
	}
}

// isCustomType проверяет, является ли тип пользовательским (не примитивным)
func isCustomType(typeName string) bool {
	primitives := map[string]bool{
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"byte": true, "rune": true,
		"float32": true, "float64": true,
		"complex64": true, "complex128": true,
		"string": true, "bool": true,
	}

	return !primitives[typeName]
}
