# Sync Photo Backend

Backend на Go для мобильного приложения синхронных фото пар. Позволяет парам делать синхронные фотографии на двух устройствах одновременно через WebSocket.

## Технологии

- **Go** 1.21+
- **PostgreSQL** 15+
- **WebSocket** (gorilla/websocket)
- **HTTP Router** (chi)
- **Database Driver** (pgx/v5)
- **Logging** (zerolog)
- **AWS S3** для хранения фото
- **JWT** для аутентификации

## Структура проекта

```
sync-photo-backend/
├── cmd/
│   └── cmd.go              # Точка входа
├── internal/
│   ├── config/             # Конфигурация
│   ├── handlers/           # HTTP/WebSocket handlers
│   ├── services/           # Бизнес-логика
│   ├── repository/         # Работа с БД
│   ├── models/             # Модели данных
│   └── middleware/         # Middleware (auth, CORS)
├── db/migrations/          # SQL миграции
├── config.yaml             # Конфигурация приложения
├── main.go                 # Главный файл
└── README.md
```

## Установка и запуск

### Требования

- Go 1.21+
- PostgreSQL 15+
- AWS S3 bucket (для хранения фото)

### 1. Клонирование и установка зависимостей

```bash
git clone <repository-url>
cd sync-photo-backend
go mod download
```

### 2. Настройка базы данных

Создайте базу данных PostgreSQL:

```sql
CREATE DATABASE syncphoto;
```

Примените миграции:

```bash
# Используйте psql или другой инструмент для выполнения миграций
psql -U postgres -d syncphoto -f db/migrations/001_init.up.sql
```

### 3. Настройка конфигурации

Отредактируйте `config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  dbname: "syncphoto"
  sslmode: "disable"

aws:
  region: "us-east-1"
  s3_bucket: "syncphoto-uploads"
  
jwt:
  secret: "your-secret-key-change-in-production"

log:
  level: "debug"
```

### 4. Настройка AWS S3

1. Создайте S3 bucket в AWS
2. Настройте AWS credentials через:
   - AWS CLI: `aws configure`
   - Environment variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
   - IAM role (если запускается на EC2)

### 5. Запуск приложения

```bash
go run main.go
```

Или скомпилируйте и запустите:

```bash
go build -o api main.go
./api
```

Сервер запустится на `http://localhost:8080`

## API Endpoints

### POST /api/v1/users
Создание анонимного пользователя.

**Запрос:**
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json"
```

**Ответ:**
```json
{
  "id": "uuid",
  "code": "ABC123",
  "token": "jwt-token",
  "created_at": "2025-01-15T10:00:00Z"
}
```

### POST /api/v1/pairs
Создание пары между двумя пользователями.

**Запрос:**
```bash
curl -X POST http://localhost:8080/api/v1/pairs \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"partner_code": "ABC123"}'
```

**Ответ:**
```json
{
  "id": "uuid",
  "user_a_id": "uuid",
  "user_b_id": "uuid",
  "created_at": "2025-01-15T10:00:00Z"
}
```

### DELETE /api/v1/pairs/:pair_id
Удаление пары.

**Запрос:**
```bash
curl -X DELETE http://localhost:8080/api/v1/pairs/:pair_id \
  -H "Authorization: Bearer <token>"
```

**Ответ:** `204 No Content`

### GET /api/v1/photos
Получение списка фото пары.

**Запрос:**
```bash
curl -X GET "http://localhost:8080/api/v1/photos?limit=50&offset=0" \
  -H "Authorization: Bearer <token>"
```

**Ответ:**
```json
{
  "photos": [...],
  "total": 150
}
```

### POST /api/v1/photos/upload
Получение pre-signed URL для загрузки фото в S3.

**Запрос:**
```bash
curl -X POST http://localhost:8080/api/v1/photos/upload \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"filename": "photo.jpg", "content_type": "image/jpeg"}'
```

**Ответ:**
```json
{
  "upload_url": "https://s3.amazonaws.com/...",
  "photo_id": "uuid",
  "expires_in": 300
}
```

## WebSocket API

### Подключение

```
ws://localhost:8080/ws?token=<jwt-token>
```

### Сообщения от клиента

#### trigger_photo
Инициация синхронного фото.

```json
{
  "type": "trigger_photo",
  "timestamp": 1705315200000
}
```

#### photo_uploaded
Подтверждение загрузки фото.

```json
{
  "type": "photo_uploaded",
  "photo_id": "uuid",
  "s3_url": "https://..."
}
```

### Сообщения от сервера

#### take_photo
Команда сделать фото (отправляется обоим пользователям).

```json
{
  "type": "take_photo",
  "initiator_id": "uuid",
  "timestamp": 1705315200000
}
```

#### partner_status
Статус партнера (онлайн/оффлайн).

```json
{
  "type": "partner_status",
  "online": true
}
```

#### error
Ошибка.

```json
{
  "type": "error",
  "message": "Partner is offline"
}
```

## Тестирование

См. файл [TESTING.md](TESTING.md) для подробных примеров curl команд.


## Разработка

### Запуск миграций

```bash
# Применить миграции
psql -U postgres -d syncphoto -f db/migrations/001_init.up.sql

# Откатить миграции
psql -U postgres -d syncphoto -f db/migrations/001_init.down.sql
```

### Логирование

Логи настраиваются через `config.yaml`. Уровни:
- `debug` - детальные логи
- `info` - информационные логи
- `warn` - предупреждения
- `error` - только ошибки

## Архитектура

Приложение использует трехслойную архитектуру:

1. **Handlers** - обработка HTTP/WebSocket запросов, валидация
2. **Services** - бизнес-логика, координация
3. **Repository** - работа с БД

WebSocket Hub управляет соединениями и синхронизацией между партнерами.

## Безопасность

- JWT токены для аутентификации (срок жизни 365 дней)
- Валидация всех входящих данных
- CORS настроен для MVP (разрешает все origins)
- Pre-signed URLs для безопасной загрузки в S3

## Лицензия

MIT
