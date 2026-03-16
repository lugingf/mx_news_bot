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
- `ENABLE_HOST_GATEWAY=true`

Подробности и nginx-пример: [deploy/README.md](/Users/evgenylugin/go/src/_pet/mx_news_bot/deploy/README.md).
