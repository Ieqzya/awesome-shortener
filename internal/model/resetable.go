package model

// generate:reset
// ResetableStruct - пример структуры для демонстрации работы генератора Reset
type ResetableStruct struct {
	I     int
	Str   string
	StrP  *string
	S     []int
	M     map[string]string
	Child *ResetableStruct
}

// generate:reset
// SimpleStruct - простая структура с примитивными типами
type SimpleStruct struct {
	ID      int
	Name    string
	Active  bool
	Score   float64
	Counter uint
}
