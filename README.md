# Trip service

HTTP-сервис поездок. Контракт — `contracts/openapi/trip-service.openapi.yaml`,
сгенерированный из него код — `internal/generated/api.gen.go`. Лабораторная
работа 1 подняла рабочий сервис с тремя ручками; лабораторная работа 2
переложила тот же функционал по слоям, закрыла тестами и линтером, добавила
логи/метрики/трейсы через OpenTelemetry — новых ручек и миграций в ЛР2 нет.

## Запуск

1. Поднимите окружение курса и подключитесь к нему: `tripgoctl cluster start`,
   `tripgoctl environment start`, `tripgoctl connect`.
2. Скопируйте `.env.example` в `.env` (или используйте созданный `tripgoctl`),
   экспортируйте переменные: `set -a; source .env; set +a`.
3. Накатите схему: `make migrate`.
4. Запустите сервис: `make run`.

Проверка: `curl -i localhost:8080/health`; `/ready` дополнительно проверяет
соединение с PostgreSQL. Сборка и статические проверки: `make build`,
`make lint`, `go vet ./...`.

## Конфигурация

Обязательны `HTTP_ADDR`, `DATABASE_URL`, `LOG_LEVEL`, `SHUTDOWN_TIMEOUT`,
`DATABASE_MAX_CONNS`, `DATABASE_MIN_CONNS`, `DATABASE_CONNECT_TIMEOUT`,
`DATABASE_QUERY_TIMEOUT`, `DATABASE_MAX_CONN_LIFETIME`,
`OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_ENDPOINT`,
`OTEL_EXPORTER_OTLP_PROTOCOL` (`http/protobuf` или `grpc`),
`OTEL_RESOURCE_ATTRIBUTES`, `OTEL_TRACES_SAMPLER` (`always_on`, `always_off`,
`parentbased_always_on` или `parentbased_always_off`). Значения и примеры — в
`.env.example`; без любой из этих переменных сервис не стартует.

Необязательные HTTP-таймауты: `HTTP_READ_TIMEOUT` (по умолчанию `5s`),
`HTTP_READ_HEADER_TIMEOUT` (`5s`), `HTTP_WRITE_TIMEOUT` (`10s`) и
`HTTP_IDLE_TIMEOUT` (`1m`).

## Архитектура

Код разложен по слоям с зависимостями внутрь: `http`/`repository` знают про
`usecase`, `usecase` знает про `domain`, `domain` не знает ни про кого. Правило
проверяет `depguard` из `.golangci.yml`.

```
cmd/trip-service/        main — читает конфиг, зовёт container.New, запускает, ждёт сигнал
internal/
  config/                разбор и валидация env
  container/             единственное место с pgxpool.New и конструкторами — собирает весь граф
  domain/trip/            сущность Trip: NewTrip (инварианты), Finish (переход статуса), ошибки
  model/                 Trip как плоский DTO между usecase/repository/http
  usecase/trip/
    create/ get/ finish/  contract.go (интерфейсы на стороне потребителя) + usecase.go
  http/
    trip/create|get|finish/  тонкие хендлеры, свой contract.go на usecase-интерфейс
    domainerror/            Problem(err) — единственное место маппинга ошибки в HTTP-ответ
    middleware/             Recover, Route (http.route + active_requests), Logging
    httputil/               общие WriteJSON/WriteProblem/ToAPI, чтобы три хендлера не дублировали код
    router.go               собирает api.ServerInterface из трёх хендлеров
  repository/trip/        реализация интерфейсов usecase поверх pgx
  infra/
    postgres/              pgxpool + Executor(ctx) — читает транзакцию из контекста или берёт пул
      transaction/          TxManager
    telemetry/              OTel SDK (трейсы+метрики), slog с trace_id/span_id, бизнес-метрики
  generated/               сгенерированный по OpenAPI код
```

Интерфейсы объявлены в `contract.go` того пакета, который их использует
(`tripRepository` — в `usecase/trip/create`, не рядом с `repository/trip`):
usecase формулирует, что ему нужно, а реализация подставляется в `container`.

### Принятые решения и отступления от шаблона задания

- **`api/` → `internal/generated/`.** Задание ЛР1 (`docs/conventions.md`, п.8)
  держало сгенерированный код в `api/` на верхнем уровне, но таблица структуры
  самой ЛР2 явно показывает `internal/generated/`. Следую более специфичному —
  текущему — указанию; генератор (`make generate`) и импорт обновлены.
- **Завершение поездки — через `SELECT ... FOR UPDATE`, а не условный
  `UPDATE`.** В ЛР1 атомарность завершения держалась на
  `UPDATE ... WHERE status = 'active'`. Чтобы `Trip.Finish()` стал настоящим
  путём перехода (а не демонстрационным методом сбоку), `usecase/trip/finish`
  теперь: блокирует строку (`GetForUpdate` в транзакции) → прогоняет переход
  через `domain.Trip.Finish()` в памяти → сохраняет (`Update`). Блокировка
  строки даёт ту же гарантию от гонки, что и условный `UPDATE`, но бизнес-правило
  живёт в домене и покрыто чистым unit-тестом, как и требует п.4.3 задания.
- **Метрики и бизнес-события — в HTTP-хендлере, а не в usecase.** `IncCreated`/
  `IncCompleted` и логи «поездка создана/завершена» вызываются в
  `http/trip/create|finish`, а не внутри usecase: усложнять usecase
  инфраструктурным интерфейсом ради события, которое и так видно из его
  результата (`created bool`, успешный `Finish`), не нужно — граница usecase
  остаётся чистой (`domain`+`repository`+`tx`), а не-доменные side-эффекты
  собраны в транспорте.
- **`http.server.active_requests` и `http.route` — через
  `api.ChiServerOptions.Middlewares`, а не через внешний `http.Handler`-враппер.**
  `oapi-codegen`-роутер резолвит `chi`-паттерн маршрута только внутри обёртки
  конкретной ручки; внешний враппер над всем `mux` в принципе не может узнать
  паттерн (запрос чи роутится уже внутри вызова `next.ServeHTTP`). Поэтому
  `middleware.Route` зарегистрирован через `Middlewares` в
  `api.ChiServerOptions` — там `chi.RouteContext(r.Context()).RoutePattern()`
  уже заполнен, а симметричные подписи increment/decrement для up-down counter
  получаются автоматически. Он же дописывает `http.route` в `otelhttp.Labeler`,
  которым пользуются встроенные метрики `otelhttp` (`http.server.request.duration`)
  и которую читает внешний `middleware.Logging` для поля `route` в логе.
  `Recover` и `Logging` остаются внешними враппer-ами над всем `mux`: им нужно
  видеть вообще каждый запрос, включая несматченные (404), которые до
  `ChiServerOptions.Middlewares` просто не доходят.
- **`db.collection.name` на метрике `db.client.operation.duration` не
  проставлен.** `otelpgx` v0.13.0 (последняя доступная) заполняет этот атрибут
  только для `CopyFrom`; для обычных `Query`/`Exec` библиотека не даёт хука для
  имени таблицы ни на спане, ни на метрике. Ручной парсинг SQL в каждом методе
  репозитория явно запрещён самим заданием («иначе половину запросов забудут
  обернуть»), поэтому оставляю `db.operation.name` (есть, через
  `otelpgx.WithMetricOperationName`) без `db.collection.name` — это
  задокументированное ограничение библиотеки, а не недосмотр.
- **Интеграционные тесты репозитория — без `testcontainers-go`.** Окружение и
  так поднимается `tripgoctl environment start`; тащить вторую независимую
  систему подъёма Postgres ради тех же интеграционных тестов добавляло бы
  зависимость без выигрыша. Тесты (`//go:build integration`) читают
  `DATABASE_URL` из процессного окружения и падают с понятным `t.Skip`, если
  её нет.

## Observability

Сервис отправляет трейсы и метрики по OTLP в коллектор (адрес — из
`OTEL_EXPORTER_OTLP_ENDPOINT`), а тот раскладывает их по Prometheus/Grafana и
Jaeger; в коде этих имён нет, только OpenTelemetry. Если коллектор недоступен,
сервис продолжает отвечать — экспорт просто логирует ошибку и повторяет на
следующем цикле.

- **Логи** — `log/slog`, JSON, в stdout. Каждая запись внутри спана несёт
  `trace_id`/`span_id` (`internal/infra/telemetry.NewLogger`); у каждой —
  `service`. Одна запись `request handled` на HTTP-запрос (в `middleware.Logging`,
  уровень `Error` только для `5xx`) плюс отдельные бизнес-события `trip created`/
  `trip finished` с `trip_id`.
- **Трейсы** — корневой спан на HTTP-запрос (`otelhttp`, имя `{method} {route}`),
  дочерние спаны на каждый поход в БД (`otelpgx`, на уровне пула — ни один метод
  репозитория не знает о трейсинге).
- **Метрики** — `http.server.request.duration`, `http.server.active_requests`,
  `db.client.operation.duration` (все — по семантическим конвенциям
  OpenTelemetry) и бизнесовые `tripgo.trips.created`/`tripgo.trips.completed`.

Порядок HTTP-middleware: `Recover → Tracing (otelhttp) → Logging → Route (внутри
роутера, на уровне сматченной ручки) → хендлер`. `Recover` ловит панику, пишет
стектрейс и отдаёт `500 internal_error`, не роняя процесс.

Проверить вручную: `tripgoctl connect` печатает адреса Jaeger/Grafana/Prometheus;
открыть трейс по `trace_id` из лога, найти `http_server_request_duration_seconds`
и `tripgo_trips_created_total`/`tripgo_trips_completed_total` в Prometheus.

## Тесты

| Что | Чем | Где |
|---|---|---|
| Домен (`NewTrip`, `Finish`) | table-driven, без моков | `internal/domain/trip` |
| Usecase | `gomock` на репозиторий/`TxManager` | `internal/usecase/trip/*` |
| Маппинг ошибок | table-driven | `internal/http/domainerror` |
| Хендлеры | `httptest` + мок usecase | `internal/http/trip/*` |
| Репозиторий | реальный PostgreSQL, тег `integration` | `internal/repository/trip` |

`make test` — unit-тесты с `-race` и покрытием `usecase`+`domain` (печатается
итогом, фактически ≥85%). `make test-integration` — интеграционные тесты
(нужен поднятый `tripgoctl environment` и `DATABASE_URL` в окружении). Моки
генерируются `go generate ./...` (`mockgen`) и коммитятся вместе с кодом.

## Решения

Создание поездки выполняется в одной транзакции: сначала добавляется запись в
`trips`, затем — начальная запись в `trip_status_history`. Транзакционный
менеджер помещает `pgx.Tx` в `context`; репозиторий сам выбирает транзакцию из
контекста или пул для обычных чтений. Вложенный `Do` использует уже открытую
транзакцию. При ошибке либо панике выполняется rollback (покрыто интеграционным
тестом `TestRepository_Create_RollsBackOnError`).

Используется изоляция PostgreSQL `Read Committed`: для этой задачи она
достаточна, поскольку запрет второй активной поездки закреплён частичным
уникальным индексом `trips_active_driver_idx`, а завершение блокирует строку
явным `SELECT ... FOR UPDATE` перед переходом статуса в домене (см.
«Архитектура» выше). Конкурентные запросы создания дают один успех и
`driver_busy`; конкурентные finish — один успех и `trip_completed`.

`/health` не обращается к БД и показывает, что процесс жив. `/ready` делает
`Ping` с таймаутом запроса и возвращает `503`, пока БД недоступна. Сервис
останавливается по `SIGINT`/`SIGTERM` через `http.Server.Shutdown`, ожидая
текущие запросы не дольше `SHUTDOWN_TIMEOUT`, затем закрывает телеметрию и пул.

### Идемпотентность

`POST /api/v1/trips` поддерживает необязательный UUID-заголовок
`Idempotency-Key`. Ключ и SHA-256 нормализованного тела резервируются в той же
транзакции, что и поездка. Повтор с тем же телом возвращает исходную поездку с
`200`; другое тело — `409 idempotency_conflict`. Первичный ключ таблицы делает
поведение безопасным при параллельных запросах. Записи хранятся 24 часа и
удаляются при следующем запросе с ключом.

## Docker

`deploy/Dockerfile` — многостадийная сборка; итоговый образ distroless,
запускается пользователем `nonroot` и содержит только статический бинарный
файл. Проверенный локально образ `trip-service-lab:local` занимает **13.1 MB**
(linux/amd64); размер может немного отличаться на другой платформе.

Для воспроизводимого запуска сервиса вместе с PostgreSQL и миграциями:

```bash
docker compose -f deploy/compose.yaml up --build
curl -i http://localhost:8080/ready
```

Контейнер `migrate` однократно накатывает миграции Goose, после чего стартует
`trip-service`. `OTEL_EXPORTER_OTLP_ENDPOINT` в `compose.yaml` указывает на
`otel-collector`, которого в самом `compose.yaml` нет — при локальном запуске
только через Docker Compose (без `tripgoctl`) экспорт телеметрии не найдёт
коллектор и будет лишь логировать ошибку, сервис при этом продолжит работать.
Чтобы остановить окружение и удалить данные PostgreSQL:

```bash
docker compose -f deploy/compose.yaml down --volumes
```

## Опрос устаревших поездок

Сканер → ограниченная приоритетная очередь → фиксированный пул воркеров → Push Service. Сканер периодически читает активные поездки из PostgreSQL порциями, используя `(COALESCE(last_position_at, started_at), id)` как ключ продолжения. Первыми обрабатываются поездки, по которым дольше всего нет координат. Повторная постановка уже ожидающей или выполняющейся задачи исключена. При сохранении координаты ожидающая задача удаляется из очереди.

При заполнении очереди более старая задача вытесняет самую свежую; в противном случае новая задача отбрасывается. Оба случая учитываются в `tripgo.stale_checks{result="dropped"}` и журналируются. После первого переполнения сканирование прекращается до следующего тика. Данные о поездках остаются в БД, поэтому на следующем тике они снова будут рассмотрены.

| Переменная | Значение по умолчанию | Назначение |
|---|---|---|
| `POSITION_CHECK_INTERVAL` | `30s` | период сканирования |
| `POSITION_STALE_AFTER` | `60s` | возраст последней координаты или старта |
| `POSITION_WORKERS` | `4` | число воркеров |
| `POSITION_QUEUE_SIZE` | `100` | вместимость очереди |
| `POSITION_SCAN_BATCH_SIZE` | `100` | размер порции SQL |
| `PUSH_HTTP_URL` | задаётся окружением | адрес Push Service |
| `PUSH_REQUEST_TIMEOUT` | `1s` | таймаут запроса к Push Service |

При остановке HTTP-сервер завершает запросы, сканер останавливается, воркеры опустошают очередь, затем закрываются клиент и пул БД. Метрики очереди, воркеров и результатов экспортируются через OpenTelemetry. Проверка индекса и пример плана: [docs/lab3-explain.txt](docs/lab3-explain.txt).
