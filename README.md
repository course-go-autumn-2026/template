# Trip Service

HTTP API на Go: поездки, PostgreSQL, pgx, chi, squirrel.

## Запуск

1. `tripgoctl cluster start && tripgoctl environment start && tripgoctl connect`
2. `cp .env.example .env`, вписать `DATABASE_URL` из `connect`
3. `make migrate`
4. `make run`

## Переменные окружения

| Переменная | Назначение |
|---|---|
| HTTP_ADDR | адрес HTTP-сервера |
| LOG_LEVEL | уровень логов |
| SHUTDOWN_TIMEOUT | таймаут graceful shutdown |
| DATABASE_URL | строка подключения к PostgreSQL |
| DATABASE_MAX_CONNS | максимум соединений в пуле |
| DATABASE_MIN_CONNS | минимум соединений в пуле |
| DATABASE_MAX_CONN_LIFETIME | время жизни соединения |
| DATABASE_CONNECT_TIMEOUT | таймаут подключения |
| DATABASE_QUERY_TIMEOUT | таймаут запроса |

## Решения

- **Уровень изоляции** — `ReadCommitted`. Для `create` и `finish` достаточно; уникальность активной поездки держит частичный уникальный индекс, а не уровень изоляции; `Serializable` дал бы лишние ретраи без выгоды.
- **Менеджер транзакций** — `txmanager.Do` открывает `pgx.Tx`, кладёт в контекст, вызывает `fn`. Ошибка или паника → `ROLLBACK`. Вложенный `Do` переиспользует транзакцию из контекста. Репозиторий достаёт исполнителя через `FromContext` — вне `Do` работает через пул.
- **Запрет двух активных поездок** — `CREATE UNIQUE INDEX trips_driver_active_uniq ON trips (driver_id) WHERE status = 'active'`. Ошибка `23505` превращается в `ErrDriverBusy` → `409 driver_busy`. Проверка `SELECT` перед `INSERT` не годится — гонка.
- **Завершение** — условный `UPDATE ... WHERE status = 'active'`. Ноль строк → доп. чтение, чтобы отличить `404` от `409`. `finished_at` не перезаписывается.