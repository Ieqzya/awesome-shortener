#!/bin/bash

# Скрипт для сборки приложения с информацией о версии

set -e

# Получаем информацию о версии
VERSION=${VERSION:-$(git describe --tags --always 2>/dev/null || echo "dev")}
DATE=$(date -u +"%Y-%m-%d")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Выводим информацию
echo "Building shortener..."
echo "  Version: $VERSION"
echo "  Date:    $DATE"
echo "  Commit:  $COMMIT"
echo ""

# Собираем приложение
go build -ldflags "\
  -X main.buildVersion=${VERSION} \
  -X main.buildDate=${DATE} \
  -X main.buildCommit=${COMMIT}" \
  -o shortener ./cmd/shortener

echo "Build complete: ./shortener"
