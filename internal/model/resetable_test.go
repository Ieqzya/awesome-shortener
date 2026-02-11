package model

import (
	"testing"
)

func TestResetableStruct_Reset(t *testing.T) {
	// Создаем структуру с заполненными полями
	str := "test string"
	child := &ResetableStruct{
		I:   42,
		Str: "child",
	}

	rs := &ResetableStruct{
		I:     100,
		Str:   "hello",
		StrP:  &str,
		S:     []int{1, 2, 3, 4, 5},
		M:     map[string]string{"key1": "value1", "key2": "value2"},
		Child: child,
	}

	// Сохраняем capacity слайса
	originalCap := cap(rs.S)

	// Вызываем Reset
	rs.Reset()

	// Проверяем примитивы
	if rs.I != 0 {
		t.Errorf("I не сброшен: ожидали 0, получили %d", rs.I)
	}

	if rs.Str != "" {
		t.Errorf("Str не сброшен: ожидали пустую строку, получили %q", rs.Str)
	}

	// Проверяем указатель на строку
	if rs.StrP == nil {
		t.Error("StrP не должен быть nil после Reset")
	} else if *rs.StrP != "" {
		t.Errorf("*StrP не сброшен: ожидали пустую строку, получили %q", *rs.StrP)
	}

	// Проверяем слайс
	if len(rs.S) != 0 {
		t.Errorf("Длина слайса S не сброшена: ожидали 0, получили %d", len(rs.S))
	}

	if cap(rs.S) != originalCap {
		t.Errorf("Capacity слайса S изменился: ожидали %d, получили %d", originalCap, cap(rs.S))
	}

	// Проверяем мапу
	if len(rs.M) != 0 {
		t.Errorf("Мапа M не очищена: ожидали длину 0, получили %d", len(rs.M))
	}

	// Проверяем вложенную структуру
	if rs.Child == nil {
		t.Error("Child не должен быть nil после Reset")
	} else {
		if rs.Child.I != 0 {
			t.Errorf("Child.I не сброшен: ожидали 0, получили %d", rs.Child.I)
		}
		if rs.Child.Str != "" {
			t.Errorf("Child.Str не сброшен: ожидали пустую строку, получили %q", rs.Child.Str)
		}
	}
}

func TestResetableStruct_Reset_Nil(t *testing.T) {
	var rs *ResetableStruct
	// Не должно паниковать
	rs.Reset()
}

func TestSimpleStruct_Reset(t *testing.T) {
	ss := &SimpleStruct{
		ID:      123,
		Name:    "Test Name",
		Active:  true,
		Score:   99.5,
		Counter: 42,
	}

	ss.Reset()

	if ss.ID != 0 {
		t.Errorf("ID не сброшен: ожидали 0, получили %d", ss.ID)
	}

	if ss.Name != "" {
		t.Errorf("Name не сброшен: ожидали пустую строку, получили %q", ss.Name)
	}

	if ss.Active != false {
		t.Errorf("Active не сброшен: ожидали false, получили %v", ss.Active)
	}

	if ss.Score != 0 {
		t.Errorf("Score не сброшен: ожидали 0, получили %f", ss.Score)
	}

	if ss.Counter != 0 {
		t.Errorf("Counter не сброшен: ожидали 0, получили %d", ss.Counter)
	}
}

func TestSimpleStruct_Reset_Nil(t *testing.T) {
	var ss *SimpleStruct
	// Не должно паниковать
	ss.Reset()
}

// Бенчмарк для проверки производительности Reset
func BenchmarkResetableStruct_Reset(b *testing.B) {
	rs := &ResetableStruct{
		I:    100,
		Str:  "hello",
		S:    make([]int, 0, 100),
		M:    make(map[string]string),
		Child: &ResetableStruct{},
	}

	// Заполняем слайс и мапу
	for i := 0; i < 50; i++ {
		rs.S = append(rs.S, i)
		rs.M[string(rune(i))] = "value"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.Reset()
		// Восстанавливаем данные для следующей итерации
		for j := 0; j < 50; j++ {
			rs.S = append(rs.S, j)
			rs.M[string(rune(j))] = "value"
		}
	}
}
