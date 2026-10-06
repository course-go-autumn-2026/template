## Запуск и локальная разработка

### 1. Подготовка базы данных

Запустите службу PostgreSQL:
```bash
sudo service postgresql start 

# 2.  Создайте пользователя и базу данных tripgo:
sudo -u postgres psql -c "CREATE USER tripgo WITH PASSWORD 'tripgo';"
sudo -u postgres psql -c "CREATE DATABASE tripgo OWNER tripgo;"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE tripgo TO tripgo;" 
# Примените схему миграций:
sed '/-- +goose Down/,$d' migrations/20260928104550_init_trips_schema.sql | psql "postgres://tripgo:tripgo@localhost:5432/tripgo?sslmode=disable"
# Для генерации типов и интерфейсов по спецификации OpenAPI выполните:
make generate 
# запуск сервиса
HTTP_ADDR=":8080" \
DATABASE_URL="postgres://tripgo:tripgo@localhost:5432/tripgo?sslmode=disable" \
go run ./cmd/trip-service 

