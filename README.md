# MX News Bot

## Deploy (GitHub Actions)

Проект деплоится blue-green в директорию `/opt/mx_news_bot`.

### GitHub Secrets

- `AWS_SSH_HOST`
- `AWS_SSH_USER`
- `AWS_SSH_PORT`
- `AWS_SSH_PRIVATE_KEY`
- `GHCR_USERNAME`
- `GHCR_TOKEN`
- `DEPLOY_CONFIG_JSON`

### GitHub Variables

- `APP_NAME=mx_news_bot`
- `APP_DIR=/opt/mx_news_bot`
- `PUBLIC_PORT=8585`
- `PUBLIC_BIND_ADDR=127.0.0.1`
- `DEPLOY_NETWORK=mx_news_bot_net` (или свой)
- `WEBHOOK_PATH=/webhook-mx`
- `METRICS_PATH=/metrics-mx`
- `ENABLE_HOST_GATEWAY=true` (если DB на host), `false` если DB в `mx_news_bot_net`

Подробности и nginx-пример: [deploy/README.md](/Users/evgenylugin/go/src/_pet/mx_news_bot/deploy/README.md).

## Отдельный запуск БД (в другой директории, отдельно от app)

1. Создайте общую сеть (один раз):
   `docker network create mx_news_bot_net || true`
2. Поднимите только БД из файла:
   `deploy/db/docker-compose.db.yml`
3. Для blue-green деплоя приложения оставьте:
   - `DEPLOY_NETWORK=mx_news_bot_net`
   - `ENABLE_HOST_GATEWAY=false` (если БД в той же docker network)
4. В `DEPLOY_CONFIG_JSON` строка подключения должна быть контейнерной:
   `database.conn = "host=db-mx port=5432 user=mx dbname=mx sslmode=disable password=mxpassword"`
