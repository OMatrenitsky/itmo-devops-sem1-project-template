#!/bin/bash
set -e

echo "Запуск приложения."
go run main.go &

# Проверка запуска

for i in {1..10}; do
  if curl -s http://localhost:8080 &> /dev/null; then
    echo "Сервер готов к работе."
    exit 0
  fi
  echo "Сервер пока не готов..."
  sleep 4
done

echo "Ошибка: сервер не запустился вовремя."
exit 1
