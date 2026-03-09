#!/bin/bash

# Скрипт для генерации Go кода из proto файлов

set -e

echo "Генерация gRPC кода из proto файлов..."

# Проверяем наличие protoc
if ! command -v protoc &> /dev/null; then
    echo "Ошибка: protoc не установлен"
    echo ""
    echo "Установите protoc:"
    echo "  macOS:   brew install protobuf"
    echo "  Linux:   sudo apt-get install protobuf-compiler"
    echo "  Windows: https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

# Проверяем наличие protoc-gen-go
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Установка protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# Проверяем наличие protoc-gen-go-grpc
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Установка protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Создаем директорию для сгенерированных файлов
mkdir -p internal/grpc/pb

# Генерируем код
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/shortener.proto

# Перемещаем сгенерированные файлы в правильную директорию
mv proto/shortener.pb.go proto/shortener_grpc.pb.go internal/grpc/pb/

echo ""
echo "✓ Код успешно сгенерирован в internal/grpc/pb/"
