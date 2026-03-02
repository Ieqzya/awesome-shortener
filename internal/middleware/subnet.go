// Package middleware содержит HTTP middleware для обработки запросов.
package middleware

import (
	"net"
	"net/http"
)

// TrustedSubnetMiddleware проверяет, что IP-адрес клиента находится в доверенной подсети.
//
// Middleware извлекает IP-адрес из заголовка X-Real-IP и проверяет его принадлежность
// к указанной подсети в формате CIDR. Если подсеть не указана (пустая строка),
// доступ запрещается для всех запросов.
//
// Параметры:
//   - trustedSubnet: подсеть в формате CIDR (например, "192.168.1.0/24")
//
// Возвращает middleware функцию, которая:
//   - Возвращает 403 Forbidden, если IP не в доверенной подсети
//   - Возвращает 403 Forbidden, если trustedSubnet пустая строка
//   - Пропускает запрос дальше, если IP в доверенной подсети
func TrustedSubnetMiddleware(trustedSubnet string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если доверенная подсеть не указана, запрещаем доступ
			if trustedSubnet == "" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Парсим доверенную подсеть
			_, subnet, err := net.ParseCIDR(trustedSubnet)
			if err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Получаем IP из заголовка X-Real-IP
			realIP := r.Header.Get("X-Real-IP")
			if realIP == "" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Парсим IP-адрес
			ip := net.ParseIP(realIP)
			if ip == nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Проверяем, входит ли IP в доверенную подсеть
			if !subnet.Contains(ip) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// IP в доверенной подсети, пропускаем запрос дальше
			next.ServeHTTP(w, r)
		})
	}
}
