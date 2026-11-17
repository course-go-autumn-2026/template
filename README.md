# Лабораторная работа 3

## Запуск

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect
make migrate
make run
```

## Тесты и проверка кода

```bash
make build
make generate
make lint
make test
make test-integration
```

## Координаты и маршрут

Добавлены сохранение координаты через `POST /api/v1/trips/{tripId}/positions`
и получение маршрута через `GET /api/v1/trips/{tripId}/positions`.
Точка и `last_position_at` сохраняются в одной транзакции через `TxManager`.
Запоздавшие координаты не уменьшают `last_position_at` благодаря SQL `GREATEST`.
Маршрут сортируется по `recorded_at`, затем `id`.

## Опрос устаревших поездок

```
Сканер по тикеру -> ограниченная очередь -> воркеры -> Push Service
```

Сканер выбирает из БД активные поездки без свежих координат, начиная с самых
старых. Если координат ещё нет, время считается от `started_at`.
Для выборки добавлен частичный индекс `trips_active_position_idx`.

| Переменная | По умолчанию |
|---|---|
| `POSITION_CHECK_INTERVAL` | `30s` |
| `POSITION_STALE_AFTER` | `60s` |
| `POSITION_WORKERS` | `4` |
| `POSITION_QUEUE_SIZE` | `100` |
| `PUSH_HTTP_URL` | из `.env` |
| `PUSH_REQUEST_TIMEOUT` | `1s` |

Повторы задач отсеиваются под mutex. После сохранения координаты ожидающая
задача отменяется. При заполнении очереди сканер не блокируется, задача получает
`dropped`, увеличиваются метрики и записывается Warn-лог.
Гейтвей отправляет `REQUEST_POSITION` через HTTP с таймаутом и без ретраев.

При остановке сначала завершаются HTTP-запросы, затем останавливается сканер,
воркеры дорабатывают очередь и закрываются ресурсы. Общий бюджет задаёт
`SHUTDOWN_TIMEOUT`.

Размер очереди, занятые воркеры и результаты обработки видны в метриках OTel.
Обработка задачи связана со сканером через span link, вызов Push Service имеет
отдельный спан.
