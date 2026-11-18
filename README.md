# TripGo: Trip Service

### 1. Запуск окружения

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect
```

### 2. Применение миграций
Примените схему базы данных:

```bash
make migrate
```

### 3. Запуск сервиса
```bash
make run
```
Сервис поднимется на порту из переменной `HTTP_ADDR`, по умолчанию `:8080`.

---

## Переменные окружения

Все параметры конфигурации передаются через переменные окружения или локальный файл `.env`:

| Переменная | По умолчанию | Описание |
|---|---|---|
| `HTTP_ADDR` | `:8080` | Адрес и порт HTTP-сервера |
| `HTTP_READ_TIMEOUT` | `5s` | Таймаут на чтение запроса целиком |
| `HTTP_READ_HEADER_TIMEOUT` | `2s` | Таймаут на чтение заголовков запроса |
| `HTTP_WRITE_TIMEOUT` | `5s` | Таймаут на запись ответа |
| `HTTP_IDLE_TIMEOUT` | `30s` | Таймаут простоя keep-alive соединения |
| `LOG_LEVEL` | `info` | Уровень логирования: `debug`, `info`, `warn`, `error` |
| `SHUTDOWN_TIMEOUT` | `10s` | Бюджет времени на graceful shutdown |
| `DATABASE_URL` | обязательная | DSN строка подключения к PostgreSQL |
| `DATABASE_MAX_CONNS` | `10` | Максимальное количество соединений в пуле `pgxpool` |
| `DATABASE_MIN_CONNS` | `2` | Минимальное количество поддерживаемых соединений в пуле |
| `DATABASE_MAX_CONN_LIFETIME` | `30m` | Максимальное время жизни одного соединения |
| `DATABASE_CONNECT_TIMEOUT` | `5s` | Таймаут на установку соединения с БД |
| `DATABASE_QUERY_TIMEOUT` | `3s` | Таймаут на выполнение одного SQL-запроса |
| `OTEL_SERVICE_NAME` | `trip-service` | Имя сервиса для телеметрии |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:22318` | Адрес OTLP-приемника телеметрии |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | Протокол экспорта OTLP: `grpc`, `http/protobuf`, `http/json` |
| `OTEL_RESOURCE_ATTRIBUTES` | `service.namespace=tripgo...` | Дополнительные атрибуты ресурса OpenTelemetry |
| `OTEL_TRACES_SAMPLER` | `parentbased_always_on` | Сэмплирование трейсов: `always_on`, `always_off`, `traceidratio`, `parentbased_always_on`, `parentbased_always_off`, `parentbased_traceidratio` |

---

## Команды Makefile

* `make run` — запуск сервиса локально.
* `make migrate` — накат всех миграций БД.
* `make migrate-down` — откат последней миграции.
* `make generate` — генерация типов и серверных интерфейсов из OpenAPI контракта, генерация моков `mockgen`.
* `make test` — запуск unit-тестов с расчетом покрытия `domain` и `usecase` (покрытие > 95%).
* `make test-integration` — запуск интеграционных тестов с тегом `integration` на реальном PostgreSQL.
* `make lint` — запуск `golangci-lint` с правилами `depguard`.

---

## Архитектура (Лабораторная работа 2)

Сервис разделен на слои с однонаправленными зависимостями внутрь: транспорт и репозиторий знают про usecase, usecase знает про domain, domain не знает ни про кого.

```
[cmd/main] ──> [internal/container]
                     │
       ┌─────────────┼──────────────┐
       ▼             ▼              ▼
[internal/http] [internal/usecase] [internal/repository]
       │             │              │
       │             ▼              ▼
       │     [internal/domain] [internal/model]
       │             ▲
       └─────────────┘
```

### Слои сервиса
* `cmd/main.go` — точка входа: считывает конфигурацию, создает зависимости через `container`, запускает HTTP-сервер и обрабатывает сигналы остановки `SIGINT`/`SIGTERM` с graceful shutdown.
* `internal/container/` — composition root: единственное место в сервисе, где создаются конкретные реализации (`pgxpool`, OpenTelemetry, репозитории, юзкейсы, хендлеры) и связываются между собой вручную без DI-фреймворка.
* `internal/domain/trip/` — правила поездки, валидация полей при создании, переходы состояний в методе `Finish`, доменные ошибки (`ErrTripNotFound`, `ErrTripCompleted`, `ErrDriverBusy`, `ValidationError`). Не зависит от HTTP, SQL и внешних библиотек.
* `internal/model/` — структуры данных, передаваемые между слоями.
* `internal/usecase/trip/` — сценарии бизнес-логики (`create`, `get`, `finish`). Пакет на каждое действие со своим `contract.go`.
* `internal/http/trip/` — HTTP-хендлеры (`create`, `get`, `finish`). Валидируют запросы, вызывают юзкейс и отдают JSON.
* `internal/http/domainerror/` — единое место маппинга доменных ошибок в RFC 9457 `application/problem+json`. Внутренние ошибки отдаются как 500 `internal_error` без раскрытия деталей бд.
* `internal/http/middleware/` — цепочка middleware: `Recover -> Tracing -> Logging -> Metrics -> router`.
* `internal/repository/trip/` — реализация хранилища на PostgreSQL (`pgx/v5`) и squirrel.
* `internal/infra/postgres/` — пул `pgxpool` с трейсером `otelpgx` и менеджер транзакций.
* `internal/infra/telemetry/` — настройка OpenTelemetry SDK и логгера `slog`.

### Правила зависимостей (`depguard`)
* `internal/domain` не импортирует `net/http`, `chi`, `pgx`, `squirrel`, `otel`, `api`.
* `internal/usecase` не импортирует `net/http`, `chi`, `pgx`, `squirrel`, `api`.
* `internal/http/*` не импортирует `internal/repository/*`, `internal/infra/*`.
Соблюдение правил проверяется линтером `depguard` в `.golangci.yml`.

### Где объявлены интерфейсы и почему
Интерфейсы объявляются на стороне потребителя в `contract.go`:
* `usecase/trip/create/contract.go` объявляет `tripRepository`, `txManager`, `logger` и `metrics`.
* `http/trip/create/contract.go` объявляет `useCase`.
* `http/server.go` объявляет `pinger` для ручки `/ready`.

**Почему интерфейсы там:**
1. **Инверсия зависимостей:** юзкейс диктует свои требования к репозиторию, а не подстраивается под него.
2. **Узкие интерфейсы:** каждому компоненту видны только нужные методы, что упрощает написание моков в тестах.
3. **Подготовка к gRPC (лабораторная работа 4):** при добавлении второго транспорта создается пакет `internal/grpc/` со своими контрактами к тем же юзкейсам. Домен и юзкейсы при этом не меняются.

### Наблюдаемость (Observability)
* **Логи (`slog`):** вывод в stdout в формате JSON. Каждая запись содержит `trace_id` и `span_id` через `TraceHandler`. В middleware пишется одна запись по завершении HTTP-запроса (`method`, `route`, `status`, `duration_ms`). Бизнес-события логируются в юзкейсах с указанием `trip_id` после успешного коммита транзакции.
* **Трейсы (OTel):** Корневой спан на входящий HTTP-запрос `{method} {http.route}` создается в middleware `Tracing`. Запросы к БД инструментируются на уровне пула через `otelpgx`.
* **Метрики (Prometheus):** `http.server.request.duration`, `http.server.active_requests`, `tripgo.trips.created`, `tripgo.trips.completed`. В атрибут `http.route` пишется строго шаблон маршрута (например, `/api/v1/trips/{tripId}`), а не URL с UUID, чтобы не раздувать кардинальность в Prometheus.
* **Middleware `Recover`:** перехватывает паники, логирует их со стектрейсом и возвращает клиенту 500 `internal_error`, не роняя сервис.

---

## Принятые технические решения

### 1. Уровень изоляции транзакций
Для транзакций в методе `txManager.Do` выбран уровень изоляции `ReadCommitted`. Это стандартный уровень в PostgreSQL с минимальными накладными расходами на блокировки.

Защита от аномалий обеспечивается схемой данных и логикой запросов:
1. Создание двух активных поездок предотвращается частичным уникальным индексом `WHERE status = 'active'`.
2. Завершение поездки защищено условием `WHERE status = 'active'` в запросе `UPDATE`.

Более тяжелые уровни (`RepeatableRead`, `Serializable`) не требуются и привели бы к лишним ошибкам сериализации с кодом 40001.

### 2. Менеджер транзакций
Менеджер транзакций реализован в пакете `internal/infra/postgres/transaction`. Метод `Do(ctx, fn)` открывает транзакцию `pgx.Tx` и передает её в контекст по приватному ключу `txKey{}`.

Репозиторий не принимает транзакцию аргументом: функция `transaction.GetExecutor(ctx, pool)` проверяет контекст и выбирает активную транзакцию или пул соединений.

При ошибке или панике внутри переданной функции транзакция откатывается, а при успехе фиксируется. Если внутри вызывается вложенный `Do`, менеджер повторно использует уже существующую транзакцию.

### 3. Запрет двух активных поездок на одного водителя
Ограничение задано в схеме базы данных:
```sql
CREATE UNIQUE INDEX trips_driver_active_idx ON trips (driver_id) WHERE status = 'active';
```
При попытке создать две активные поездки на одного водителя база возвращает ошибку уникальности с кодом 23505. Репозиторий перехватывает её и возвращает `ErrDriverBusy`, который хендлер преобразует в статус 409 Conflict со структурой `driver_busy`.

### 4. Защита от состояния гонки при завершении поездки
Завершение поездки выполняется атомарно через условие в SQL-запросе:
```sql
UPDATE trips
SET status = 'completed', finished_at = $1, updated_at = now()
WHERE id = $2 AND status = 'active'
RETURNING id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at, created_at, updated_at;
```
Если несколько параллельных запросов пытаются завершить одну поездку, строку обновит только один из них. Если затронуто 0 строк, репозиторий проверяет причину: если поездка уже завершена, возвращается `409 trip_completed`, если не найдена — `404 trip_not_found`.

### 5. Ручки мониторинга
* `/health` проверяет жизнеспособность процесса HTTP-сервера и возвращает 200 OK. Ручка намеренно не опрашивает базу данных, чтобы временная недоступность БД не приводила к аварийному перезапуску контейнера оркестратором.
* `/ready` проверяет готовность сервиса принимать пользовательский трафик через `pool.Ping(ctx)`. При недоступности базы возвращается 503 Service Unavailable, при доступности — 200 OK.

