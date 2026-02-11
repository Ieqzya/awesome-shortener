// Package osexit реализует анализатор, запрещающий прямой вызов os.Exit в функции main пакета main.
//
// Анализатор проверяет, что функция main пакета main не содержит прямых вызовов os.Exit.
// Это помогает обеспечить корректное завершение программы с выполнением всех defer'ов
// и правильной очисткой ресурсов.
//
// Пример некорректного кода:
//
//	package main
//
//	import "os"
//
//	func main() {
//		os.Exit(1) // Ошибка: прямой вызов os.Exit в main
//	}
//
// Пример корректного кода:
//
//	package main
//
//	import "os"
//
//	func main() {
//		exitCode := run()
//		os.Exit(exitCode)
//	}
//
//	func run() int {
//		// Основная логика программы
//		return 0
//	}
package osexit

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Analyzer - анализатор, запрещающий использование os.Exit в функции main пакета main.
var Analyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "запрещает прямой вызов os.Exit в функции main пакета main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Проверяем только пакет main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			// Ищем объявление функции
			funcDecl, ok := node.(*ast.FuncDecl)
			if !ok {
				return true
			}

			// Проверяем, что это функция main
			if funcDecl.Name.Name != "main" {
				return true
			}

			// Проверяем тело функции на наличие вызовов os.Exit
			ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
				callExpr, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Проверяем, является ли вызов селектором (package.Function)
				selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				// Проверяем, что это вызов os.Exit
				ident, ok := selExpr.X.(*ast.Ident)
				if !ok {
					return true
				}

				if ident.Name == "os" && selExpr.Sel.Name == "Exit" {
					pass.Reportf(callExpr.Pos(), "прямой вызов os.Exit в функции main запрещен")
				}

				return true
			})

			return true
		})
	}

	return nil, nil
}
