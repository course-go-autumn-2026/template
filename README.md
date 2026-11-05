# Сервис поездок (Trip Service)

Реализация лабораторных работ курса «Разработка микросервисов на Go».

## 🚀 Запуск

1. Поднимите окружение (PostgreSQL, Jaeger, Grafana, OTel Collector):
   ```bash
   tripgoctl cluster start
   tripgoctl environment start
   tripgoctl connect
2. Примените миграции к базе данных (используйте порт из вывода tripgoctl connect или файла .env):
    psql "postgres://tripgo:tripgo@localhost:22032/tripgo?sslmode=disable" -f migrations/00001_init.sql
3. Запустите сервер:
    make run
4. Тесты и линтер:
    make lint — запуск линтера (golangci-lint).
    make test — запуск unit-тестов с флагом -race и подсчетом покрытия.
    make test-integration — запуск интеграционных тестов с реальной БД.
5. Архитектура:
    Проект построен по принципам чистой архитектуры. Зависимости направлены строго внутрь: транспорт и репозитории знают про usecase, usecase знает про домен, а домен не знает ни про кого.
    internal/domain/trip — доменные правила (например, FinishTrip). Не знает ни про HTTP, ни про базу данных.
    internal/model — структуры данных (DTO), которые ходят между слоями.
    internal/usecase/trip/... — бизнес-логика (create, get, finish). Знает только про интерфейсы репозиториев, объявленные в contract.go.
    internal/http/... — транспортный слой (хендлеры). Знает про usecase и маппит доменные ошибки в HTTP-коды (domainerror).
    internal/repository/trip — реализация репозитория (pgx + squirrel).
    internal/infra — инфраструктура (транзакции, телеметрия).
    internal/container — единая точка сборки зависимостей (ручной DI).
6. Observability:
    Логи: slog в формате JSON с полями trace_id и span_id.
    Трейсы: OpenTelemetry с экспортом по HTTP (OTLP).
    Метрики: tripgo.trips.created и tripgo.trips.completed (экспортируются в Prometheus и доступны в Grafana).