# Shortlink

Сервис для сокращения ссылок на Go.

## Запуск

### PostgreSQL

```bash
docker-compose -f docker-compose.postgresql.yml up --build
```

### In-memory

```bash
docker-compose -f docker-compose.inmemory.yml up --build
```

## API

### 1. Создать короткую ссылку

`POST /api/shorten`

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url":"https://google.com"}'
```

Ответ:

```json
{"code":"{short}"}
```

### 2. Получить оригинальный URL

`GET /api/expand/{short}`

```bash
curl -X GET http://localhost:8080/api/expand/{short}
```

Ответ:

```json
{"url":"https://google.com"}
```