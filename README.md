# Trip Service

Сервис управления поездками и сбора координат водителей на Go.

## Конфигурация и запуск

Сервис настраивается через переменные окружения (значения локального окружения генерируются в `.env` командой `tripgoctl environment start`):

| Переменная | Описание | По умолчанию |
|---|---|---|
| `HTTP_ADDR` | Адрес и порт HTTP-сервера | `:8080` |
| `DATABASE_URL` | Строка подключения к PostgreSQL | *(обязательно)* |
| `LOG_LEVEL` | Уровень логирования (`debug`, `info`, `warn`, `error`) | `info` |
| `SHUTDOWN_TIMEOUT` | Таймаут корректной остановки сервиса | `10s` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Адрес OTLP-коллектора для трейсов и метрик | `localhost:4318` |
| `POSITION_CHECK_INTERVAL` | Период тикера сканера (пауза между повторными пушами) | `30s` |
| `POSITION_STALE_AFTER` | Время без координат, после которого поездка считается устаревшей | `60s` |
| `POSITION_WORKERS` | Количество воркеров в пуле | `4` |
| `POSITION_QUEUE_SIZE` | Размер ограниченной очереди задач | `100` |
| `PUSH_HTTP_URL` | Базовый URL Push Service (из `.env`) | `http://localhost:8081` |
| `PUSH_REQUEST_TIMEOUT` | Таймаут запроса в Push Service | `1s` |

```zsh
tripgoctl cluster start
tripgoctl environment start 
tripgoctl connect
set -a; source .env; set +a
make migrate
make run
```

## Опрос устаревших поездок

### Схема работы

```text
PostgreSQL (trips) --> Сканер (Ticker) --> Очередь (chan) --> Воркеры (Pool) --> Push Service
                                                ^
                                                |  (отмена при приходе координаты)
                                     POST /trips/{id}/positions
```

1. **Сканер:** одна горутина с тикером раз в `POSITION_CHECK_INTERVAL` выбирает активные поездки, по которым координат не было дольше `POSITION_STALE_AFTER` (считается в SQL по индексу `idx_trips_active_stale`).
2. **Очередь:** буферизованный канал ёмкостью `POSITION_QUEUE_SIZE` с дедупликацией в памяти (`inFlight`), чтобы одна поездка не попадала в очередь дважды. Если приходит координата, поездка снимается с опроса.
3. **Воркеры и Push Service:** фиксированный пул из `POSITION_WORKERS` горутин читает задачи из канала и отправляет `REQUEST_POSITION` в Push Service с таймаутом `PUSH_REQUEST_TIMEOUT`.

### Что происходит при исчерпании очереди

Запись в очередь выполняется неблокирующим `select` с веткой `default`. Если очередь заполнена (`POSITION_QUEUE_SIZE` достигнут), задача отбрасывается (`dropped`), сканер не блокируется и продолжает работу, инкрементируются метрики `tripgo.stale_checks{result="dropped"}` и `tripgo.position_requests{result="dropped"}`, а в лог пишется предупреждение (`Warn`). Поездка повторно попадёт в выборку на следующем тике сканера.