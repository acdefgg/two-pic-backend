# Тестирование API

## Базовый URL
```
http://localhost:8080
```

## 1. Создание пользователя (POST /api/v1/users)

Создает нового анонимного пользователя и возвращает код, токен и ID.

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json"
```

**Ответ:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "code": "A3B7K9",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "created_at": "2025-01-15T10:00:00Z"
}
```

**Сохраните токен для следующих запросов:**
```bash
export TOKEN1="ваш_токен_пользователя_1"
export USER1_ID="id_пользователя_1"
export USER1_CODE="код_пользователя_1"
```

Создайте второго пользователя:
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json"
```

```bash
export TOKEN2="ваш_токен_пользователя_2"
export USER2_ID="id_пользователя_2"
export USER2_CODE="код_пользователя_2"
```

---

## 2. Создание пары (POST /api/v1/pairs)

Создает пару между текущим пользователем и партнером по коду.

```bash
curl -X POST http://localhost:8080/api/v1/pairs \
  -H "Authorization: Bearer $TOKEN1" \
  -H "Content-Type: application/json" \
  -d '{
    "partner_code": "'$USER2_CODE'"
  }'
```

**Ответ:**
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440000",
  "user_a_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_b_id": "660e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-01-15T10:05:00Z"
}
```

**Сохраните ID пары:**
```bash
export PAIR_ID="id_пары"
```

**Ошибки:**
- `400` - Неверный формат partner_code
- `401` - Не авторизован
- `404` - Партнер не найден
- `409` - Пользователь уже в паре или партнер уже в паре

---

## 3. Получение списка фото (GET /api/v1/photos)

Получает список всех фото в паре пользователя.

```bash
curl -X GET "http://localhost:8080/api/v1/photos?limit=50&offset=0" \
  -H "Authorization: Bearer $TOKEN1"
```

**Query параметры:**
- `limit` (опционально, по умолчанию 50, максимум 100)
- `offset` (опционально, по умолчанию 0)

**Ответ:**
```json
{
  "photos": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "pair_id": "660e8400-e29b-41d4-a716-446655440000",
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "s3_url": "https://syncphoto-uploads.s3.us-east-1.amazonaws.com/...",
      "taken_at": "2025-01-15T10:10:00Z",
      "created_at": "2025-01-15T10:10:05Z"
    }
  ],
  "total": 1
}
```

**Ошибки:**
- `401` - Не авторизован
- `404` - Пользователь не в паре

---

## 4. Получение pre-signed URL для загрузки фото (POST /api/v1/photos/upload)

Получает pre-signed URL для загрузки фото в S3.

```bash
curl -X POST http://localhost:8080/api/v1/photos/upload \
  -H "Authorization: Bearer $TOKEN1" \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "photo.jpg",
    "content_type": "image/jpeg"
  }'
```

**Ответ:**
```json
{
  "upload_url": "https://s3.amazonaws.com/syncphoto-uploads/...",
  "photo_id": "880e8400-e29b-41d4-a716-446655440000",
  "expires_in": 300
}
```

**Сохраните photo_id:**
```bash
export PHOTO_ID="photo_id"
export UPLOAD_URL="pre_signed_url"
```

**Загрузка фото в S3:**
```bash
curl -X PUT "$UPLOAD_URL" \
  -H "Content-Type: image/jpeg" \
  --data-binary @/path/to/your/photo.jpg
```

**Ошибки:**
- `400` - Неверный формат запроса
- `401` - Не авторизован
- `404` - Пользователь не в паре

---

## 5. Удаление пары (DELETE /api/v1/pairs/:pair_id)

Удаляет пару (разрыв пары).

```bash
curl -X DELETE "http://localhost:8080/api/v1/pairs/$PAIR_ID" \
  -H "Authorization: Bearer $TOKEN1"
```

**Ответ:**
```
204 No Content
```

**Ошибки:**
- `401` - Не авторизован
- `403` - Пользователь не является членом пары
- `404` - Пара не найдена

---

## 6. WebSocket подключение

### Подключение

```bash
# Используйте wscat или другой WebSocket клиент
wscat -c "ws://localhost:8080/ws?token=$TOKEN1"
```

Или через JavaScript:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?token=' + TOKEN1);

ws.onopen = () => {
  console.log('Connected');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};

ws.onerror = (error) => {
  console.error('Error:', error);
};

ws.onclose = () => {
  console.log('Disconnected');
};
```

### Сообщения от клиента к серверу

#### Инициация синхронного фото (trigger_photo)
```json
{
  "type": "trigger_photo",
  "timestamp": 1705315200000
}
```

**Отправка через wscat:**
```bash
{"type":"trigger_photo","timestamp":1705315200000}
```

#### Подтверждение загрузки фото (photo_uploaded)
```json
{
  "type": "photo_uploaded",
  "photo_id": "880e8400-e29b-41d4-a716-446655440000",
  "s3_url": "https://syncphoto-uploads.s3.us-east-1.amazonaws.com/pair_id/photo_id.jpg"
}
```

**Отправка через wscat:**
```bash
{"type":"photo_uploaded","photo_id":"880e8400-e29b-41d4-a716-446655440000","s3_url":"https://..."}
```

### Сообщения от сервера к клиенту

#### Команда сделать фото (take_photo)
```json
{
  "type": "take_photo",
  "initiator_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": 1705315200000
}
```

#### Статус партнера (partner_status)
```json
{
  "type": "partner_status",
  "online": true
}
```

или

```json
{
  "type": "partner_status",
  "online": false
}
```

#### Ошибка (error)
```json
{
  "type": "error",
  "message": "Partner is offline"
}
```

---

## Полный пример тестирования

```bash
#!/bin/bash

BASE_URL="http://localhost:8080"

echo "1. Создание первого пользователя..."
RESPONSE1=$(curl -s -X POST $BASE_URL/api/v1/users \
  -H "Content-Type: application/json")
echo $RESPONSE1 | jq

TOKEN1=$(echo $RESPONSE1 | jq -r '.token')
USER1_CODE=$(echo $RESPONSE1 | jq -r '.code')
USER1_ID=$(echo $RESPONSE1 | jq -r '.id')

echo ""
echo "2. Создание второго пользователя..."
RESPONSE2=$(curl -s -X POST $BASE_URL/api/v1/users \
  -H "Content-Type: application/json")
echo $RESPONSE2 | jq

TOKEN2=$(echo $RESPONSE2 | jq -r '.token')
USER2_CODE=$(echo $RESPONSE2 | jq -r '.code')
USER2_ID=$(echo $RESPONSE2 | jq -r '.id')

echo ""
echo "3. Создание пары..."
PAIR_RESPONSE=$(curl -s -X POST $BASE_URL/api/v1/pairs \
  -H "Authorization: Bearer $TOKEN1" \
  -H "Content-Type: application/json" \
  -d "{\"partner_code\": \"$USER2_CODE\"}")
echo $PAIR_RESPONSE | jq

PAIR_ID=$(echo $PAIR_RESPONSE | jq -r '.id')

echo ""
echo "4. Получение pre-signed URL для загрузки фото..."
UPLOAD_RESPONSE=$(curl -s -X POST $BASE_URL/api/v1/photos/upload \
  -H "Authorization: Bearer $TOKEN1" \
  -H "Content-Type: application/json" \
  -d '{"filename": "photo.jpg", "content_type": "image/jpeg"}')
echo $UPLOAD_RESPONSE | jq

PHOTO_ID=$(echo $UPLOAD_RESPONSE | jq -r '.photo_id')
UPLOAD_URL=$(echo $UPLOAD_RESPONSE | jq -r '.upload_url')

echo ""
echo "5. Получение списка фото..."
curl -s -X GET "$BASE_URL/api/v1/photos?limit=50&offset=0" \
  -H "Authorization: Bearer $TOKEN1" | jq

echo ""
echo "6. WebSocket подключение..."
echo "Используйте: wscat -c \"ws://localhost:8080/ws?token=$TOKEN1\""
echo "Затем отправьте: {\"type\":\"trigger_photo\",\"timestamp\":$(date +%s)000}"
```

---

## Установка wscat (для тестирования WebSocket)

```bash
npm install -g wscat
```

## Тестирование с помощью HTTPie

Если предпочитаете HTTPie вместо curl:

```bash
# Создание пользователя
http POST localhost:8080/api/v1/users

# Создание пары
http POST localhost:8080/api/v1/pairs \
  Authorization:"Bearer $TOKEN1" \
  partner_code="$USER2_CODE"

# Получение фото
http GET localhost:8080/api/v1/photos \
  Authorization:"Bearer $TOKEN1" \
  limit==50 offset==0

# Получение pre-signed URL
http POST localhost:8080/api/v1/photos/upload \
  Authorization:"Bearer $TOKEN1" \
  filename="photo.jpg" \
  content_type="image/jpeg"

# Удаление пары
http DELETE localhost:8080/api/v1/pairs/$PAIR_ID \
  Authorization:"Bearer $TOKEN1"
```
