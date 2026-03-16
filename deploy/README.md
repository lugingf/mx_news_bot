# Deploy Notes (`mx_news_bot`)

`mx_news_bot` deploys blue-green to `/opt/mx_news_bot`.

## Required GitHub Secrets

- `AWS_SSH_HOST`
- `AWS_SSH_USER`
- `AWS_SSH_PORT`
- `AWS_SSH_PRIVATE_KEY`
- `GHCR_USERNAME`
- `GHCR_TOKEN`
- `DEPLOY_CONFIG_JSON` (runtime `config.json` content)

## Required GitHub Variables

- `APP_NAME=mx_news_bot`
- `APP_DIR=/opt/mx_news_bot`
- `PUBLIC_PORT=8585`
- `PUBLIC_BIND_ADDR=127.0.0.1`
- `DEPLOY_NETWORK=mx_news_bot_net` (or custom)
- `WEBHOOK_PATH=/webhook-mx`
- `METRICS_PATH=/metrics-mx` (recommended)
- `ENABLE_HOST_GATEWAY=true` (recommended if DB is on host)

## Nginx (shared host with `astro_mind_bot`)

```nginx
server {
    listen 443 ssl;
    server_name lugingfwebhookambot.com;

    ssl_certificate /etc/letsencrypt/live/lugingfwebhookambot.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/lugingfwebhookambot.com/privkey.pem;

    location = /webhook-path {
        proxy_pass http://127.0.0.1:8443;
    }

    location = /webhook-mx {
        proxy_pass http://127.0.0.1:8585;
    }
}
```

Use HTTP for local upstream ports (`8443`, `8585`), because TLS is terminated by host nginx.
