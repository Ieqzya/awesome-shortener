#!/bin/bash

# Скрипт для генерации самоподписанного TLS сертификата

set -e

CERT_FILE="cmd/shortener/cert.pem"
KEY_FILE="cmd/shortener/key.pem"

echo "Генерация самоподписанного TLS сертификата..."
echo ""

# Генерируем сертификат
openssl req -x509 -newkey rsa:2048 \
  -keyout "$KEY_FILE" \
  -out "$CERT_FILE" \
  -days 365 \
  -nodes \
  -subj "/CN=localhost"

echo ""
echo "Сертификат успешно создан:"
echo "  Сертификат: $CERT_FILE"
echo "  Ключ:       $KEY_FILE"
echo ""
echo "Для запуска сервера с HTTPS используйте:"
echo "  ./shortener -s"
echo "  или"
echo "  ENABLE_HTTPS=true ./shortener"
