# Trip Service — Лабораторная работа 2

Сервис управления поездками (Trip Service), реализованный на Go в рамках курса по микросервисам.
В рамках Лабораторной работы 2 сервис переведен на чистую слоистую архитектуру (Clean Architecture), покрыт тестами (unit с `gomock`, интеграционные на PostgreSQL), полностью инструментирован OpenTelemetry (трейсы, метрики, структурированные логи `slog`) и проверен строгим набором линтеров (`golangci-lint` с `depguard`).

---

## 1. Запуск сервиса

### Требования
- Go 1.24+
- `tripgoctl` (или запущенные PostgreSQL, OTLP collector, Jaeger, Prometheus)

### Команды
```bash
# 1. Запуск окружения (PostgreSQL + Observability stack)
tripgoctl environment start
tripgoctl connect

# 2. Накатить миграции
make migrate

# 3. Запустить линтер
make lint

# 4. Прогнать unit-тесты с race detector и отчетом о покрытии
make test

# 5. Прогнать интеграционные тесты с реальной базой
make test-integration

# 6. Запустить сервис локально
make run

# 7. Запустить кодогенерацию (OpenAPI + gomock)
make generate
```

---

## 2. Архитектура сервиса

### 2.1. Раскладка слоев (Clean Architecture)

В сервисе соблюдаются два ключевых правила:
1. **Пакет на действие, а не файл на слой** — у каждого юзкейса (`create`, `get`, `finish`) и у каждого HTTP-эндпоинта свой изолированный пакет со своим `contract.go`, логикой и тестами.
2. **Инфраструктура отделена от бизнес-логики** — база данных (`pgx`), телеметрия (OTel), логгер (`slog`) вынесены в `internal/infra/`.

```
cmd/
  trip-service/
    main.go                   # Composition root: читает конфиг, зовет container.New, ждет сигналов SIGINT/SIGTERM
internal/
  config/                     # Чтение и строгая валидация переменных окружения
  container/                  # Единственное место создания реализаций и связывания зависимостей (DI)
  domain/                     # Чистое ядро бизнес-логики (не зависит от net/http, SQL, pgx, chi, OTel)
    trip/                     # Модель поездки, методы перехода состояний (Finish), валидация, доменные ошибки
  model/                      # Cross-layer структуры данных (CreateTripInput, TripStatusHistory)
  usecase/                    # Прикладные сценарии приложения
    trip/
      create/                 # Создание поездки: usecase.go, contract.go, usecase_test.go, mocks_test.go
      get/                    # Получение поездки: usecase.go, contract.go, usecase_test.go, mocks_test.go
      finish/                 # Завершение поездки: usecase.go, contract.go, usecase_test.go, mocks_test.go
  http/                       # Транспортный уровень (HTTP / REST)
    server.go                 # Адаптер роутера, реализующий api.ServerInterface
    trip/
      create/                 # HTTP-хендлер POST /api/v1/trips: handler.go, convert.go, mocks_test.go
      get/                    # HTTP-хендлер GET /api/v1/trips/{tripId}
      finish/                 # HTTP-хендлер POST /api/v1/trips/{tripId}/finish
    domainerror/              # Единый маппинг «доменная ошибка -> RFC 9457 Problem JSON»
    middleware/               # Recover -> Tracing -> Logging -> Metrics
  repository/
    trip/                     # PostgreSQL-репозиторий: repo.go, save.go, fetch.go, convert.go, contract.go
  infra/
    postgres/                 # Пул соединений pgxpool с инструментированием otelpgx
      transaction/            # Менеджер транзакций TxManager с поддержкой вложенности и rollback on panic
    telemetry/                # Инициализация OpenTelemetry (TracerProvider, MeterProvider, OTLP) и slog
  generated/                  # Код, сгенерированный oapi-codegen по контракту OpenAPI
migrations/                   # SQL-миграции goose
```

### 2.2. Направление зависимостей и правила depguard

Зависимости строго направлены **внутрь**:
- `internal/http` зависит от `internal/usecase` и `internal/domain`. Транспортный слой **не имеет доступа** к `repository` и `infra`.
- `internal/usecase` зависит от `internal/domain` и работает с внешними зависимостями только через **интерфейсы**, объявленные в `contract.go`. Usecase не знает про `net/http`, `chi`, `pgx`, `squirrel`.
- `internal/domain` — независимое ядро, не импортирующее внешние библиотеки и инфраструктуру.

Правила проверяются автоматически линтером `depguard` при каждом `make lint`.

### 2.3. Где объявлены интерфейсы зависимостей и почему

Интерфейсы зависимостей объявлены **на стороне потребителя (consumer-driven interfaces)** в неэкспортируемом виде:
- `internal/usecase/trip/create/contract.go` объявляет `tripRepository` (только методы `Create` и `CreateStatusHistory`) и `txManager`.
- `internal/usecase/trip/get/contract.go` объявляет `tripRepository` (только метод `GetByID`).
- `internal/usecase/trip/finish/contract.go` объявляет `tripRepository` (методы `Finish` и `CreateStatusHistory`) и `txManager`.
- `internal/http/trip/*/contract.go` объявляет интерфейс `usecase` с методом `Execute`.

**Обоснование:**
1. **Принцип разделения интерфейса (Interface Segregation Principle, ISP)**: каждый юзкейс требует ровно тот набор методов, который ему необходим для работы.
2. **Простота тестирования**: mock-генератор (`mockgen`) создает минимальный мок под конкретный пакет без необходимости мокать десятки неиспользуемых методов единого репозитория.
3. **Готовность ко 2-му транспорту (gRPC в ЛР4)**: когда в лабораторной работе 4 появится gRPC-транспорт (`internal/grpc/`), юзкейсы останутся нетронутыми! gRPC-сервер просто вызовет те же самые `usecase.Execute()`, передав доменные структуры.

### 2.4. Единый маппинг ошибок (`domainerror`)

Все доменные ошибки объявлены в `internal/domain/trip/errors.go`:
- `ErrTripNotFound`
- `ErrTripCompleted`
- `ErrDriverBusy`
- `*ValidationError`

Централизованный маппинг расположен в `internal/http/domainerror`:
- Доменные ошибки транслируются в RFC 9457 `api.Problem`.
- Хендлеры не занимаются `switch err`: они передают ошибку юзкейса в `domainerror.Write(w, err)`.
- **Безопасность**: при возникновении внутренних ошибок БД/драйвера клиенту отдается статус `500` с кодом `internal_error` и нейтральным текстом `"Internal server error"`. Стектрейсы, SQL-запросы и детали СУБД **наружу не утекают**, а фиксируются в логах.

---

## 3. Observability (OpenTelemetry, логи и метрики)

### 3.1. Цепочка Middleware (фиксированный порядок)
```
Recover → Tracing → Logging → Metrics → Роутер (Chi)
```

1. **Recover**: перехватывает паники через `recover()`, логирует стек через `logger.ErrorContext`, отдает клиенту `500 internal_error`, предотвращая падение процесса.
2. **Tracing**: извлекает контекст W3C TraceContext из HTTP-заголовков (`traceparent`), открывает корневой спан `{method} {http.route}` (например, `POST /api/v1/trips`).
3. **Logging**: пишет ровно **одну** структурированную JSON-запись на завершение запроса с атрибутами `method`, `route`, `status`, `duration_ms`, `trace_id`, `span_id`. Уровень `Info` для `2xx`/`4xx`, `Error` для `5xx`. Тела запроса/ответа не логируются.
4. **Metrics**: измеряет время ответа и ведет счетчик активных запросов.

### 3.2. Метрики

| Метрика | Тип | Единица | Лейблы | Назначение |
|---|---|---|---|---|
| `http.server.request.duration` | Histogram | s | `http.request.method`, `http.route`, `http.response.status_code` | Время ответа HTTP |
| `http.server.active_requests` | UpDownCounter | {request} | `http.request.method`, `http.route` | Количество запросов в обработке |
| `db.client.operation.duration` | Histogram | s | `db.operation.name`, `db.collection.name`, `error.type` | Время запросов в PostgreSQL (через `otelpgx`) |
| `tripgo.trips.created` | Counter | {trip} | — | Количество созданных поездок |
| `tripgo.trips.completed` | Counter | {trip} | — | Количество завершенных поездок |

> В `http.route` записывается только шаблон маршрута (например, `/api/v1/trips/{tripId}`), что предотвращает взрывной рост кардинальности метрик (metric cardinality explosion).

### 3.3. Корреляция логов с трейсами
В `internal/infra/telemetry` реализован `contextHandler` для `slog.Handler`. При логировании с контекстом (`logger.InfoContext(ctx, ...)` или через middleware) логгер автоматически извлекает активный спан из контекста и добавляет поля:
- `"trace_id"`: 32-значный hex-идентификатор трейса
- `"span_id"`: 16-значный hex-идентификатор спана
По `trace_id` из лога можно мгновенно найти соответствующий трейс в UI Jaeger.

---

## 4. Вопросы к защите

1. **Где объявлены интерфейсы зависимостей и почему там?**
   - Интерфейсы объявлены на стороне потребителя в `contract.go` каждого пакета. Это соответствует Interface Segregation Principle и Dependency Inversion Principle.
   - При добавлении gRPC в ЛР4 юзкейсы менять не потребуется: слой gRPC будет лишь еще одним клиентом юзкейсов.
2. **Пример unit-теста и какую регрессию он поймает:**
   - Тест `TestUsecase_Execute/driver_busy_returns_ErrDriverBusy` проверяет реакцию юзкейса на конфликт занятости водителя. Он unit-тест, так как изолирован от БД через мок `tripRepository`. Он поймает регрессию, если кто-то сломает проброс доменной ошибки или проигнорирует ошибку репозитория.
3. **Как `trace_id` попадает в лог:**
   - Middleware `Tracing` стартует спан и помещает его в `r.Context()`.
   - Обертка `contextHandler` в `telemetry/logger.go` перехватывает каждую запись, вызывает `trace.SpanFromContext(ctx)`, и если спан валиден, инжектирует атрибуты `trace_id` и `span_id`.
