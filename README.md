# SIM Rent — аренда сим-карт

Фронтенд (Vite) + бекенд (Go/Gin + PostgreSQL).

## Структура

```
project/
├── index.html        # Фронтенд
├── style.css
├── vite.config.js    # Прокси /api → localhost:8080
└── backend/          # Go-бекенд
    ├── main.go
    ├── go.mod
    ├── .env
    ├── db_connection/
    ├── intern/
    │   ├── DTO/
    │   ├── auth/
    │   ├── domain/
    │   ├── http/
    │   ├── repository/
    │   └── service/
    ├── migrations/
    └── router/
```

## Запуск бекенда

```bash
cd backend
go run main.go
```

Бекенд слушает порт `:8080`. Перед запуском:

1. Установите PostgreSQL и создайте базу.
2. Примените миграции из `backend/migrations/` (файлы `*.up.sql`).
3. Настройте `backend/.env`:
   - `DB_CONN` — строка подключения к PostgreSQL
   - `JWT_SECRET` — секрет для подписи JWT
   - `JWT_EXPIRE_HOURS` — срок жизни токена в часах

## Запуск фронтенда

```bash
npm install
npm run dev
```

Vite проксирует все запросы `/api/*` на `http://localhost:8080`.

## API

| Метод  | Путь                  | Доступ  | Описание                      |
|--------|-----------------------|---------|-------------------------------|
| POST   | /api/login            | публичный | Вход, возвращает JWT         |
| POST   | /api/register         | публичный | Регистрация пользователя     |
| GET    | /api/cards/active     | авторизованный | Активные аренды         |
| GET    | /api/cards/expired    | авторизованный | История аренд          |
| POST   | /api/change-password  | авторизованный | Смена пароля           |
| POST   | /api/admin/cards      | админ | Добавить сим-карту                 |
| POST   | /api/admin/cards/rent | админ | Выдать аренду пользователю         |
