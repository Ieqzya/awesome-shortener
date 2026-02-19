package pool_test

import (
	"fmt"

	"awesome-shortener/internal/model"
	"awesome-shortener/internal/pool"
)

// ExamplePool демонстрирует базовое использование Pool
func ExamplePool() {
	// Создаем пул для структур SimpleStruct
	p := pool.New(func() *model.SimpleStruct {
		return &model.SimpleStruct{}
	})

	// Получаем объект из пула
	obj := p.Get()
	obj.ID = 123
	obj.Name = "Example"
	obj.Active = true

	fmt.Printf("Before Put: ID=%d, Name=%s, Active=%v\n", obj.ID, obj.Name, obj.Active)

	// Возвращаем в пул (автоматически вызывается Reset)
	p.Put(obj)

	// Получаем объект снова
	obj2 := p.Get()
	fmt.Printf("After Get: ID=%d, Name=%s, Active=%v\n", obj2.ID, obj2.Name, obj2.Active)

	// Output:
	// Before Put: ID=123, Name=Example, Active=true
	// After Get: ID=0, Name=, Active=false
}

// ExamplePool_withDefer демонстрирует использование Pool с defer
func ExamplePool_withDefer() {
	p := pool.New(func() *model.SimpleStruct {
		return &model.SimpleStruct{}
	})

	// Функция, использующая объект из пула
	processRequest := func(id int, name string) {
		obj := p.Get()
		defer p.Put(obj) // Гарантируем возврат в пул

		obj.ID = id
		obj.Name = name
		obj.Active = true

		// Работаем с объектом...
		fmt.Printf("Processing: ID=%d, Name=%s\n", obj.ID, obj.Name)
	}

	processRequest(1, "First")
	processRequest(2, "Second")

	// Output:
	// Processing: ID=1, Name=First
	// Processing: ID=2, Name=Second
}

// ExamplePool_nested демонстрирует использование Pool с вложенными структурами
func ExamplePool_nested() {
	p := pool.New(func() *model.ResetableStruct {
		return &model.ResetableStruct{
			S: make([]int, 0, 10),
			M: make(map[string]string),
		}
	})

	obj := p.Get()

	// Заполняем данные
	obj.I = 42
	obj.Str = "test"
	obj.S = append(obj.S, 1, 2, 3)
	obj.M["key"] = "value"

	fmt.Printf("Before: I=%d, Str=%s, len(S)=%d, len(M)=%d\n",
		obj.I, obj.Str, len(obj.S), len(obj.M))

	// Возвращаем в пул
	p.Put(obj)

	// Получаем снова
	obj2 := p.Get()
	fmt.Printf("After: I=%d, Str=%s, len(S)=%d, len(M)=%d, cap(S)=%d\n",
		obj2.I, obj2.Str, len(obj2.S), len(obj2.M), cap(obj2.S))

	// Output:
	// Before: I=42, Str=test, len(S)=3, len(M)=1
	// After: I=0, Str=, len(S)=0, len(M)=0, cap(S)=10
}
