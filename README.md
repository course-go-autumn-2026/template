# Trip Service

HTTP-сервис для создания, получения и завершения поездок. Данные хранятся в PostgreSQL.

## API

| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/api/v1/trips` | Создать поездку |
| `GET` | `/api/v1/trips/{tripId}` | Получить поездку |
| `POST` | `/api/v1/trips/{tripId}/finish` | Завершить поездку |
| `GET` | `/health` | Проверить процесс |
| `GET` | `/ready` | Проверить PostgreSQL |

Ошибки API возвращаются в формате `application/problem+json`.

## Локальный запуск

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect
make migrate
make run
```

`tripgoctl connect` создаёт `.env` с адресом локальной PostgreSQL. Значения по умолчанию находятся в `.env.example`.

## Переменные окружения

| Переменная | Назначение | Пример |
|---|---|---|
| `HTTP_ADDR` | Адрес HTTP-сервера | `:8080` |
| `HTTP_READ_TIMEOUT` | Таймаут чтения запроса | `5s` |
| `HTTP_READ_HEADER_TIMEOUT` | Таймаут чтения заголовков | `2s` |
| `HTTP_WRITE_TIMEOUT` | Таймаут записи ответа | `10s` |
| `HTTP_IDLE_TIMEOUT` | Таймаут keep-alive соединения | `60s` |
| `HTTP_MAX_HEADER_BYTES` | Максимальный размер заголовков | `1048576` |
| `SHUTDOWN_TIMEOUT` | Время graceful shutdown | `10s` |
| `LOG_LEVEL` | Уровень логирования | `info` |
| `DATABASE_URL` | Подключение к PostgreSQL | `postgres://tripgo:tripgo@localhost:5432/tripgo?sslmode=disable` |
| `DATABASE_MAX_CONNS` | Максимальный размер пула | `10` |
| `DATABASE_MIN_CONNS` | Минимальный размер пула | `2` |
| `DATABASE_MAX_CONN_LIFETIME` | Время жизни соединения | `30m` |
| `DATABASE_CONNECT_TIMEOUT` | Таймаут подключения | `5s` |
| `DATABASE_QUERY_TIMEOUT` | Таймаут операций с БД | `3s` |

Все переменные обязательны. При некорректной конфигурации сервис не запускается.

## Команды

```bash
make build         # собрать сервис
make run           # запустить сервис
make migrate       # применить миграции
make migrate-down  # откатить последнюю миграцию
make generate      # сгенерировать код из OpenAPI
make test          # запустить тесты с race detector
make check         # выполнить build и test
```

## Принятые решения

- HTTP-маршруты, транспортные типы и серверный интерфейс генерируются из OpenAPI.
- Используется уровень изоляции `ReadCommitted`: конкурентные ограничения обеспечиваются атомарными SQL-операциями и ограничениями PostgreSQL.
- Transaction manager хранит транзакцию в `context.Context`. Репозитории используют транзакцию внутри `Do` и пул вне него. Вложенный `Do` переиспользует текущую транзакцию.
- Ошибка или паника внутри `Do` приводит к rollback, успешное выполнение - к commit.
- Создание поездки и запись начального статуса выполняются в одной транзакции.
- Частичный уникальный индекс по `driver_id WHERE status = 'active'` запрещает две активные поездки одного водителя.
- Завершение выполняется через условный `UPDATE ... WHERE status = 'active'`, поэтому два конкурентных запроса не могут завершить поездку дважды.
- `/health` не обращается к БД. `/ready` выполняет `Ping` PostgreSQL с таймаутом.

## Idempotency-Key

`POST /api/v1/trips` поддерживает заголовок `Idempotency-Key` в формате UUID.

- первый запрос возвращает `201`;
- повтор с тем же ключом и телом возвращает ту же поездку и `200`;
- тот же ключ с другим телом возвращает `409 idempotency_conflict`;
- ключ хранится 24 часа.

Резервирование ключа, создание поездки и запись истории выполняются в одной транзакции.

## Docker

Итоговый образ основан на `scratch`, содержит статические бинарники сервиса и мигратора и запускается от пользователя `65532:65532`.

```bash
docker build -t trip-service:lab1 .

docker run --rm --network host \
  --env-file .env.example --env-file .env \
  trip-service:lab1 /app/migrator up

docker run --rm --network host \
  --env-file .env.example --env-file .env \
  trip-service:lab1
```

Размер итогового образа — примерно **12.1 MiB**.
