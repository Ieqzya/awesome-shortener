package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPingDatabase_NoDB(t *testing.T) {
	// Тест когда база данных не настроена
	handler := PingDatabase(nil)
	
	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Ожидали код 500, получили %d", w.Code)
	}
}
