package pool

import (
	"sync"
	"testing"
)

// TestStruct - тестовая структура с методом Reset
type TestStruct struct {
	ID      int
	Name    string
	Counter int
	Data    []byte
}

func (t *TestStruct) Reset() {
	t.ID = 0
	t.Name = ""
	t.Counter = 0
	t.Data = t.Data[:0]
}

func TestPool_New(t *testing.T) {
	pool := New(func() *TestStruct {
		return &TestStruct{
			Data: make([]byte, 0, 100),
		}
	})

	if pool == nil {
		t.Fatal("New() вернул nil")
	}
}

func TestPool_GetPut(t *testing.T) {
	pool := New(func() *TestStruct {
		return &TestStruct{
			Data: make([]byte, 0, 100),
		}
	})

	// Получаем объект из пула
	obj := pool.Get()
	if obj == nil {
		t.Fatal("Get() вернул nil")
	}

	// Проверяем начальное состояние
	if obj.ID != 0 || obj.Name != "" || obj.Counter != 0 {
		t.Error("Объект не в начальном состоянии")
	}

	// Изменяем объект
	obj.ID = 123
	obj.Name = "test"
	obj.Counter = 42
	obj.Data = append(obj.Data, []byte("hello")...)

	// Возвращаем в пул
	pool.Put(obj)

	// Получаем объект снова (может быть тот же самый)
	obj2 := pool.Get()

	// Проверяем, что объект был сброшен
	if obj2.ID != 0 {
		t.Errorf("ID не сброшен: ожидали 0, получили %d", obj2.ID)
	}
	if obj2.Name != "" {
		t.Errorf("Name не сброшен: ожидали пустую строку, получили %q", obj2.Name)
	}
	if obj2.Counter != 0 {
		t.Errorf("Counter не сброшен: ожидали 0, получили %d", obj2.Counter)
	}
	if len(obj2.Data) != 0 {
		t.Errorf("Data не сброшен: ожидали длину 0, получили %d", len(obj2.Data))
	}

	// Проверяем, что capacity сохранен
	if cap(obj2.Data) != 100 {
		t.Errorf("Capacity Data изменился: ожидали 100, получили %d", cap(obj2.Data))
	}
}

func TestPool_MultipleObjects(t *testing.T) {
	pool := New(func() *TestStruct {
		return &TestStruct{}
	})

	// Получаем несколько объектов
	obj1 := pool.Get()
	obj2 := pool.Get()
	obj3 := pool.Get()

	// Устанавливаем разные значения
	obj1.ID = 1
	obj2.ID = 2
	obj3.ID = 3

	// Возвращаем в пул
	pool.Put(obj1)
	pool.Put(obj2)
	pool.Put(obj3)

	// Получаем объекты снова
	newObj1 := pool.Get()
	newObj2 := pool.Get()
	newObj3 := pool.Get()

	// Все должны быть сброшены
	if newObj1.ID != 0 || newObj2.ID != 0 || newObj3.ID != 0 {
		t.Error("Не все объекты были сброшены")
	}
}

func TestPool_Concurrent(t *testing.T) {
	pool := New(func() *TestStruct {
		return &TestStruct{
			Data: make([]byte, 0, 100),
		}
	})

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				// Получаем объект
				obj := pool.Get()

				// Используем объект
				obj.ID = id
				obj.Counter = j
				obj.Data = append(obj.Data, byte(id))

				// Возвращаем в пул
				pool.Put(obj)
			}
		}(i)
	}

	wg.Wait()

	// Проверяем, что объекты в пуле сброшены
	obj := pool.Get()
	if obj.ID != 0 || obj.Counter != 0 || len(obj.Data) != 0 {
		t.Error("Объект не был корректно сброшен после конкурентного использования")
	}
}

func TestPool_ResetCalled(t *testing.T) {
	// Просто проверяем, что после Put объект сброшен
	pool := New(func() *TestStruct {
		return &TestStruct{
			Data: make([]byte, 0, 100),
		}
	})

	obj := pool.Get()
	obj.ID = 42
	obj.Name = "test"
	obj.Counter = 100

	// Возвращаем в пул (должен вызваться Reset)
	pool.Put(obj)

	// Получаем объект снова
	obj2 := pool.Get()

	// Проверяем, что Reset был вызван
	if obj2.ID != 0 {
		t.Errorf("ID не сброшен: ожидали 0, получили %d", obj2.ID)
	}
	if obj2.Name != "" {
		t.Errorf("Name не сброшен: ожидали пустую строку, получили %q", obj2.Name)
	}
	if obj2.Counter != 0 {
		t.Errorf("Counter не сброшен: ожидали 0, получили %d", obj2.Counter)
	}
}

// Бенчмарк для сравнения с обычным созданием объектов
func BenchmarkPool_GetPut(b *testing.B) {
	pool := New(func() *TestStruct {
		return &TestStruct{
			Data: make([]byte, 0, 1024),
		}
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := pool.Get()
			obj.ID = 123
			obj.Name = "benchmark"
			obj.Data = append(obj.Data, []byte("data")...)
			pool.Put(obj)
		}
	})
}

func BenchmarkDirect_NewObject(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := &TestStruct{
				Data: make([]byte, 0, 1024),
			}
			obj.ID = 123
			obj.Name = "benchmark"
			obj.Data = append(obj.Data, []byte("data")...)
			// Объект будет собран GC
			_ = obj
		}
	})
}

// Бенчмарк для проверки overhead от Reset
func BenchmarkPool_Reset(b *testing.B) {
	obj := &TestStruct{
		ID:      123,
		Name:    "test",
		Counter: 42,
		Data:    make([]byte, 100),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj.Reset()
	}
}
