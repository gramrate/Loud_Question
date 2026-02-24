# LoudQuestionBot

Telegram-бот с режимом игры по вопросам и админ-панелью для управления своими вопросами.

## Возможности

- Игра: получить вопрос, показать ответ, перейти к следующему.
- Админка: добавить вопрос, просмотреть свои вопросы, отредактировать, удалить.
- Главное меню через `/menu`.
- При `/start` бот отправляет приветствие и сразу показывает меню.
- Кнопка `Админка` в меню видна только пользователям из `ADMIN_IDS`.

## Логика показа вопросов

- Учет просмотра ведется **по аккаунту Telegram** (`user_id`).
- Вопрос, который пользователь уже видел, ему больше не показывается.
- Вопросы, созданные самим пользователем, ему в игре не показываются.
- При создании вопроса автор автоматически помечается как уже видевший этот вопрос.
- Для другого аккаунта этот же вопрос будет доступен (если он еще не видел его).

## Стек

- Go 1.25
- [go-telegram/bot](https://github.com/go-telegram/bot)
- PostgreSQL 16
- Redis 7
- Docker Compose

## Быстрый старт

1. Скопируйте пример переменных окружения:

```bash
cp .env.example .env
```

2. Заполните `.env`:

- `BOT_TOKEN` — токен Telegram-бота
- `ADMIN_IDS` — список Telegram `user_id` админов через запятую
- `POSTGRES_*` и `POSTGRES_DSN` — настройки Postgres
- `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB` — настройки Redis

3. Запустите проект:

```bash
docker compose up -d --build
```

4. Проверьте статус и логи:

```bash
docker compose ps
docker compose logs -f bot
```

## Команды бота

- `/start` — приветствие + показ главного меню
- `/menu` — открыть главное меню

## Полезные Docker-команды

Пересоздать контейнеры с пересборкой:

```bash
docker compose up -d --build --force-recreate
```

Полная очистка (контейнеры, образы, тома) и чистая пересборка:

```bash
docker compose down --rmi all --volumes --remove-orphans
docker builder prune -af
docker compose build --no-cache
docker compose up -d
```

## Структура проекта

- `internal/adapters/controller/telegram` — Telegram-контроллер и меню
- `internal/adapters/repository/postgres` — вопросы и история просмотренных вопросов
- `internal/adapters/repository/redisstate` — хранение состояния форм админки
- `internal/domain/service` — бизнес-логика игры/админки/доступа
- `internal/domain/schema` — доменные модели
