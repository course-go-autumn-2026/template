# TripGo — сервис управления поездками

Сервис на Go для создания, получения и завершения поездок. В качестве хранилища
используется PostgreSQL, доступ к БД — через `pgx/v5` и `squirrel`, HTTP-слой
сгенерирован из OpenAPI-контракта через `oapi-codegen`.

## Требования

- Go 1.22+
- PostgreSQL 15+
- `make` (опционально)

## Быстрый старт

1. Склонируйте репозиторий и перейдите в него:

   ```bash
   git clone <url-вашего-репозитория>
   cd <репозиторий>

2. Создайте .env на основе примера и заполните DATABASE_URL:
   cp .env.example .env

3. Примените миграции:
   
   goose -dir ./migrations postgres "$DATABASE_URL" up

4. Запустите сервис:
   make run

# Переменные окружения
| Переменная	            | Описание	                        | Пример              |
| HTTP_ADDR	               | адрес и порт                      | HTTP-сервера	:8080 |
| DATABASE_URL	            |строка подключения                 | к PostgreSQL	postgres://user:pass@localhost:5432/tripgo?sslmode=disable|
| LOG_LEVEL	               |уровень логирования: debug, info, warn, error | info |
| SHUTDOWN_TIMEOUT         |	таймаут graceful shutdown        |	10s |
| DATABASE_MAX_CONNS       |	максимум соединений в пуле       |	10  |
| DATABASE_MIN_CONNS	      | минимум соединений в пуле	      | 2    |
|DATABASE_CONNECT_TIMEOUT  |	таймаут подключения к БД	      |  5s  |
|DATABASE_QUERY_TIMEOUT	   |таймаут выполнения запроса	      |  5s  |
|DATABASE_MAX_CONN_LIFETIME|	время жизни соединения в пуле	   |1h    |
| HTTP_READ_TIMEOUT        |	таймаут чтения HTTP-запроса      |	5s  |
| HTTP_WRITE_TIMEOUT	      |таймаут записи HTTP-ответа	      |10s   |
| READY_TIMEOUT	         |таймаут проверки готовности /ready |	5s  |

#API
|Метод |	Путь                         |	Описание                   |
|POST	 |/api/v1/trips                 |	создать поездку            |
|GET	 |/api/v1/trips/{tripId}        |	получить поездку по ID     |
|POST	 |/api/v1/trips/{tripId}/finish |	завершить поездку          |
|GET	 |/health	                    | liveness-проба               |
|GET	 |/ready	                       |readiness-проба (проверяет БД)|

# Решения

# Уровень изоляции транзакций
Транзакции стартуют с уровнем ReadCommitted. Это задаётся явно в internal/transaction/pgx.go:

func (b *PGXBeginner) Begin(ctx context.Context) (Tx, error) {
	return b.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
}

Почему именно ReadCommitted:

Защита от гонок уже обеспечена на уровне схемы БД: частичный уникальный индекс trips_one_active_per_driver_idx и использование условия WHERE status = 'active' при обновлении. Этого достаточно для текущих сценариев.

Использование Serializable приводило бы к ошибкам сериализации (40001) при параллельных запросах, которые требуют сложной логики ретраев. Поскольку гонки предотвращаются на уровне БД, мы выбрали более производительный уровень без необходимости ретраев.

# Менеджер транзакций
internal/transaction.Manager — тонкая обёртка над pgxpool.Pool, которая:

1. Кладёт активную транзакцию в context (ключ txContextKey{}).

2. В WithinTransaction проверяет, есть ли уже транзакция в контексте. Если
есть — переиспользует её (вложенные вызовы не открывают новую транзакцию).

3. Если транзакции нет — начинает новую, выполняет fn, коммитит при успехе
или откатывает при ошибке/панике.

Для отката/коммита использует context.WithoutCancel, чтобы отмена
родительского контекста не помешала корректно закрыть транзакцию.

Репозитории (Trips, History) получают Querier через
transaction.TxFromContext(ctx), поэтому один и тот же код работает и внутри
транзакции, и вне её.

# Запрет двух активных поездок у одного водителя

На уровне БД создан частичный уникальный индекс:
CREATE UNIQUE INDEX trips_one_active_per_driver_idx
    ON trips (driver_id)
    WHERE status = 'active';

Он гарантирует, что у водителя не может быть больше одной поездки в статусе
active. При попытке вставить вторую активную поездку PostgreSQL вернёт
23505 (unique_violation), а репозиторий различает эту ошибку по имени
ограничения и возвращает ErrDriverBusy, который хендлер превращает в
409 Conflict.

Дополнительно в Trips.Finish используется UPDATE ... WHERE status = 'active',
чтобы завершение уже завершённой поездки не проходило молча.

# Тесты
make test
Запускает go test -race ./... — с детектором гонок, как требуют конвенции
курса.

# Структура проекта

.
├── api/                  # сгенерированный код из OpenAPI
├── cmd/                  # точка входа
├── contracts/            # OpenAPI-контракт
├── deploy/               # docker-compose и прочее для локального окружения
├── internal/
│   ├── config/           # загрузка и валидация env
│   ├── database/         # подключение к PostgreSQL
│   ├── handler/          # HTTP-хендлеры
│   ├── repository/       # работа с БД (squirrel)
│   ├── service/          # бизнес-логика
│   └── transaction/      # менеджер транзакций
├── migrations/           # goose-миграции
├── Makefile
└── README.md


