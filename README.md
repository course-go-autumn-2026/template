# Trip Service

Сервис управления поездками на Go (Clean Architecture, PostgreSQL, OpenTelemetry).

## 1. Окружение и миграции

Скопируйте `environment.toml` в корень репозитория и запустите инфраструктуру:

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect
