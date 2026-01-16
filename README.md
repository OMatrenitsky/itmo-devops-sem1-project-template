# Финальный проект 1 семестра

REST API сервис для загрузки и выгрузки данных о ценах.

## Требования к системе

Ubuntu 24.04 + PostgreSQL 16.

## Установка и запуск

1. Установите PostgreSQL (если не установлен) и настройте:

```bash
   sudo apt update
   sudo apt install -y postgresql

   sudo -u postgres psql
   CREATE USER validator WITH PASSWORD 'val1dat0r';
   CREATE DATABASE "project-sem-1" OWNER validator;
   GRANT ALL PRIVILEGES ON DATABASE "project-sem-1" TO validator;
   \q
   psql -h localhost -U validator -d "project-sem-1"
   CREATE TABLE prices (
    id TEXT,
    created_at DATE,
    name TEXT,
    category TEXT,
    price NUMERIC
   );
   ```

2. Клонирование репозитория
```
git clone git@github.com:OMatrenitsky/itmo-devops-sem1-project-template.git
cd itmo-devops-sem1-project
```
3. Подготовка окружения
Запустите скрипт для установки зависимостей и настройки базы данных:
```
chmod +x scripts/*.sh
./scripts/prepare.sh
```
4. Запуск сервера
```
./scripts/run.sh
```

## Тестирование

Запустите тесты с требуемым уровнем сложности.

Простой уровень, треубется запустить:
```
./scripts/tests.sh 1
```
Директория `sample_data` - это пример директории, которая является разархивированной версией файла `sample_data.zip`

## API Эндпоинты
#### 1. POST /api/v0/prices
Описание: `Загружает CSV-данные в базу данных`
Метод: `POST`
Параметры: `file – CSV-файл в формате ZIP-архива`
Пример запроса:
```
curl -X POST -F "file=@sample_data.zip" http://localhost:8080/api/v0/prices
```
Пример ответа (JSON):
```
{
  "total_items": 100,
  "total_categories": 15,
  "total_price": 100000
}
```

##### 2. GET /api/v0/prices
Описание: `Выгружает данные из базы в формате ZIP-архива`
Метод: `GET`
Пример запроса:
```
curl -X GET http://localhost:8080/api/v0/prices -o response.zip

```

## Контакт

- email: Oleg-Matrenitsky@yandex.ru
