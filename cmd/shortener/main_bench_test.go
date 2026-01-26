package main

import (
	"fmt"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// BenchmarkStringConcatenation сравнивает разные способы конкатенации строк
func BenchmarkStringConcatenation(b *testing.B) {
	b.Run("fmt.Sprintf", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = fmt.Sprintf("https://example%d.com/test", i)
		}
	})

	b.Run("string concatenation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = "https://example" + strconv.Itoa(i) + ".com/test"
		}
	})

	b.Run("strings.Builder", func(b *testing.B) {
		b.ReportAllocs()
		var builder strings.Builder
		for i := 0; i < b.N; i++ {
			builder.Reset()
			builder.WriteString("https://example")
			builder.WriteString(strconv.Itoa(i))
			builder.WriteString(".com/test")
			_ = builder.String()
		}
	})
}

// BenchmarkJSONCreation сравнивает способы создания JSON
func BenchmarkJSONCreation(b *testing.B) {
	url := "https://example.com/test"

	b.Run("fmt.Sprintf", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = fmt.Sprintf(`{"url":"%s"}`, url)
		}
	})

	b.Run("string concatenation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = `{"url":"` + url + `"}`
		}
	})

	b.Run("strings.Builder", func(b *testing.B) {
		b.ReportAllocs()
		var builder strings.Builder
		for i := 0; i < b.N; i++ {
			builder.Reset()
			builder.WriteString(`{"url":"`)
			builder.WriteString(url)
			builder.WriteString(`"}`)
			_ = builder.String()
		}
	})
}

// BenchmarkHTTPRequest бенчмарк HTTP запроса
func BenchmarkHTTPRequest(b *testing.B) {
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		body := strings.NewReader("https://example.com/test")
		req := httptest.NewRequest("POST", "/", body)
		w := httptest.NewRecorder()
		
		// Симулируем обработку запроса
		_ = req
		_ = w
	}
}