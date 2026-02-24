# LoudQuestionBot - Громкий вопрос

Telegram-бот с игровым режимом и админкой.

## Стек

- Go + `github.com/go-telegram/bot`
- PostgreSQL для вопросов и истории показов
- Redis для FSM состояний админ-форм
- Docker Compose для полного запуска (бот + БД)

## Запуск

1. Заполните `.env` (или скопируйте `.env.example`):

```bash
cp .env.example .env
```

2. Поднимите всё через Docker Compose:

```bash
docker compose up --build -d
```

3. Логи:

```bash
docker compose logs -f bot
```

## Переменные

- `BOT_TOKEN` - токен Telegram-бота
- `ADMIN_IDS` - список Telegram user_id админов через запятую
- `POSTGRES_DSN` - строка подключения к Postgres
- `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB` - Redis

## Архитектура

Структура сделана в стиле `Leech-ru`:

- `internal/adapters/app` - инициализация приложения
- `internal/adapters/app/service_provider` - провайдер зависимостей
- `internal/adapters/controller/telegram` - Telegram-контроллер
- `internal/adapters/repository/postgres` - репозиторий вопросов/seen
- `internal/adapters/repository/redisstate` - хранение FSM состояний
- `internal/domain/service/*` - бизнес-логика
- `internal/domain/schema` - доменные модели
