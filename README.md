# Trip Service

HTTP-сервис управления поездками на Go. Позволяет создавать, получать и завершать поездки. В ЛР2 существующий API из ЛР1 сохранён, а код разделён по слоям; добавлены тесты, линтер и наблюдаемость через OpenTelemetry.

## Возможности

| Метод | Маршрут | Назначение |
| --- | --- | --- |
| `POST` | `/api/v1/trips` | Создать поездку (`201`) |
| `GET` | `/api/v1/trips/{tripId}` | Получить поездку (`200` или `404`) |
| `POST` | `/api/v1/trips/{tripId}/finish` | Завершить поездку (`200` или `409`) |
| `GET` | `/health` | Проверить работу HTTP-сервера |
| `GET` | `/ready` | Проверить доступность PostgreSQL |

Создание поддерживает необязательный заголовок `Idempotency-Key` (UUID). Повтор с тем же ключом и телом возвращает ту же поездку (`200`), с другим телом - `409 idempotency_conflict`. Ключи хранятся 24 часа. Ошибки API представлены в формате RFC 9457 (`application/problem+json`).

**Стек:** Go 1.27, `net/http`, chi, PostgreSQL, pgx/pgxpool, Squirrel, Goose, oapi-codegen, OpenTelemetry, `slog`. Для тестов - `gomock` и `httptest`.

## Быстрый запуск

Команды выполняются из корня проекта **в Linux/WSL**. Нужны Go, Docker и `tripgoctl`.

```bash
# PostgreSQL, OTel Collector, Jaeger, Prometheus и Grafana
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect

# Подготовка базы и запуск сервера
make migrate
make run
```

HTTP API будет доступен на `http://localhost:8080`. `tripgoctl connect` показывает адрес базы и ссылки на интерфейсы наблюдаемости.

**Настройка OTLP.** В `.env.example` указан стандартный `http://localhost:4318`, но локальный порт коллектора, выданный `tripgoctl connect`, может отличаться. Если в `.env` отсутствует `OTEL_EXPORTER_OTLP_ENDPOINT`, добавьте туда актуальный HTTP-адрес OTLP из вывода `connect`, например:

```dotenv
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:22318
```

`make run` последовательно загружает `.env.example` и `.env`, поэтому локальное значение имеет приоритет. `.env` не коммитится; после пересоздания окружения адрес следует проверить заново.

## Проверка и сборка

```bash
make generate           # OpenAPI-код и моки через go:generate
make test               # все unit/HTTP-тесты с race detector; покрытие domain/usecase
make test-integration   # интеграционные тесты с PostgreSQL
make lint               # golangci-lint по .golangci.yml
make build              # сборка исполняемого файла
make docker-build       # многостадийный Docker-образ
make build VERSION=1.2.3 # service.version через -ldflags (по умолчанию dev)
make docker-build VERSION=1.2.3
```

Для `make test-integration` необходимы доступная PostgreSQL, применённые миграции и корректный `DATABASE_URL` в `.env`. Интеграционные тесты используют тег `integration`; основной набор тестов не требует запущенной базы. Скрипты Makefile используют POSIX shell.

## Архитектура

```text
.
├── cmd/trip-service/              запуск и graceful shutdown
├── internal/
│   ├── config/                    конфигурация приложения
│   ├── container/                 сборка зависимостей
│   ├── domain/trip/               модель, правила и ошибки
│   ├── usecase/trip/
│   │   ├── create/                создание поездки
│   │   ├── get/                   получение поездки
│   │   └── finish/                завершение поездки
│   ├── http/
│   │   ├── trip/{create,get,finish}/ HTTP-обработчики
│   │   ├── domainerror/           преобразование ошибок в RFC 9457
│   │   └── middleware/            Recover → Tracing → Logging → Metrics
│   ├── repository/trip/           запросы PostgreSQL
│   ├── infra/
│   │   ├── postgres/              трассировка SQL
│   │   │   └── transaction/       управление транзакциями
│   │   └── telemetry/             OpenTelemetry и JSON-логи
│   └── generated/                 код из OpenAPI (не редактировать вручную)
├── contracts/openapi/             исходный контракт API
└── migrations/                    схема базы данных
```

**Зависимости направлены внутрь.** Домен не зависит от HTTP, PostgreSQL и OTel. Usecase использует доменную модель и узкие интерфейсы из собственных `contract.go`; HTTP-обработчики также описывают свои зависимости на стороне потребителя. Конкретные реализации связываются в `internal/container`, без DI-фреймворка. Границы слоёв дополнительно контролирует `depguard`.

**Почему нет отдельного `model`.** Между слоями используется `domain.Trip`: модель поездки пока едина, поэтому промежуточная структура не добавляет полезной изоляции. При появлении различных контрактов слоёв будет оправдано выделение отдельной модели.

**Транзакции и конкурентность.** Менеджер в `internal/infra/postgres/transaction` выполняет `COMMIT` или `ROLLBACK`. Создание поездки и записи в истории статусов атомарно. Частичный уникальный индекс не позволяет водителю иметь более одной активной поездки; завершение выполняется условным `UPDATE`. Идемпотентность основана на резервировании ключа в транзакции и сравнении SHA-256 хэша тела запроса. Контракт OpenAPI и миграции в ЛР2 не изменялись.

Usecase проверяет допустимость завершения через `CanFinish`, а PostgreSQL атомарно меняет статус и задаёт `finished_at`. `now()` внутри одной транзакции даёт согласованное время поездки и истории. `COMMIT` использует контекст запроса с таймаутом; `ROLLBACK` — `context.WithoutCancel` с таймаутом. Неудачный откат при panic логируется, исходная panic передаётся дальше.

## Тестирование

- **Домен:** table-driven тесты правил и переходов статусов.
- **Usecase:** unit-тесты с `gomock`; моки генерируются рядом с интерфейсами в `mocks_test.go`.
- **HTTP:** `httptest`, проверка валидации, ошибок и middleware.
- **Репозиторий:** интеграционные тесты на реальном PostgreSQL, включая откат транзакции, идемпотентность и конкурентное создание поездок.

Целевой порог покрытия `internal/domain/...` и `internal/usecase/...` - не ниже 50%. Запуск тестов и линтера выполняется в Linux/WSL.

`make test` запускает тесты всех пакетов, а `-coverpkg` ограничивает измерение покрытия пакетами domain/usecase. Итог выводится из `coverage.out`.

## Логи, метрики и трейсы

На каждый HTTP-запрос пишется структурированный JSON-лог через `slog`: метод, шаблон маршрута, статус, длительность, `trace_id` и `span_id`. Тела запросов и персональные идентификаторы не используются как метки метрик. Порядок middleware: **Recover → Tracing → Logging → Metrics → Router**.

Трейсы HTTP и запросов PostgreSQL отправляются по OTLP в коллектор; для pgx используется `otelpgx`. Данные можно просматривать в **Jaeger**, а метрики - в **Prometheus/Grafana**. Адреса выводит `tripgoctl connect`. Недоступность коллектора не должна останавливать HTTP API.

OTel SDK читает `OTEL_TRACES_SAMPLER` и `OTEL_TRACES_SAMPLER_ARG`; например, `parentbased_traceidratio` и `0.1` включают выборку 10% корневых трейсов. `OTEL_METRIC_EXPORT_INTERVAL` задаётся в миллисекундах (`10000` в `.env.example`). Ошибки OTel направляются в JSON-лог с уровнем Warn. Гистограммы длительности используют секунды и границы от `0.005` до `10`. Шаблоны маршрутов берутся из chi, поэтому UUID не попадают в метки.

Основные метрики:

| OpenTelemetry | В Prometheus |
| --- | --- |
| `http.server.request.duration` | `http_server_request_duration_seconds_*` |
| `http.server.active_requests` | `http_server_active_requests` |
| `db.client.operation.duration` | `db_client_operation_duration_seconds_*` |
| `tripgo.trips.created` | `tripgo_trips_created_total` |
| `tripgo.trips.completed` | `tripgo_trips_completed_total` |

Для быстрой проверки отправьте `POST /api/v1/trips`, затем откройте в Jaeger трейс запроса и убедитесь, что в нём есть вложенные SQL-спаны. В Grafana → Explore → Prometheus проверьте:

```promql
http_server_request_duration_seconds_count
tripgo_trips_created_total
tripgo_trips_completed_total
```

Результаты ручной проверки: [трейсы Jaeger](docs/screenshots/Jaeger/) и [метрики Grafana](docs/screenshots/Grafana/).

## Дополнительно

- Docker-образ собирается в несколько стадий; итоговый контейнер основан на `distroless` и работает от `nonroot`.
- При `SIGINT` или `SIGTERM` сервер завершает приём запросов, выполняет graceful shutdown и закрывает пул соединений PostgreSQL.
