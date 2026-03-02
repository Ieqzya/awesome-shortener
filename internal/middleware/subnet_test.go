package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrustedSubnetMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		trustedSubnet  string
		realIP         string
		expectedStatus int
	}{
		{
			name:           "IP в доверенной подсети",
			trustedSubnet:  "192.168.1.0/24",
			realIP:         "192.168.1.10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IP не в доверенной подсети",
			trustedSubnet:  "192.168.1.0/24",
			realIP:         "10.0.0.1",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Пустая доверенная подсеть",
			trustedSubnet:  "",
			realIP:         "192.168.1.10",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Отсутствует заголовок X-Real-IP",
			trustedSubnet:  "192.168.1.0/24",
			realIP:         "",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Некорректный IP",
			trustedSubnet:  "192.168.1.0/24",
			realIP:         "invalid-ip",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Некорректная подсеть",
			trustedSubnet:  "invalid-subnet",
			realIP:         "192.168.1.10",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "IPv6 в доверенной подсети",
			trustedSubnet:  "2001:db8::/32",
			realIP:         "2001:db8::1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IPv6 не в доверенной подсети",
			trustedSubnet:  "2001:db8::/32",
			realIP:         "2001:db9::1",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Localhost в подсети 127.0.0.0/8",
			trustedSubnet:  "127.0.0.0/8",
			realIP:         "127.0.0.1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Оборачиваем в middleware
			middleware := TrustedSubnetMiddleware(tt.trustedSubnet)(handler)

			// Создаем тестовый запрос
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}

			// Создаем recorder для записи ответа
			rr := httptest.NewRecorder()

			// Выполняем запрос
			middleware.ServeHTTP(rr, req)

			// Проверяем статус
			if rr.Code != tt.expectedStatus {
				t.Errorf("Ожидали статус %d, получили %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}
