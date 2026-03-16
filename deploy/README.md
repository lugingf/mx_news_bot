# Deploy Notes

`mx_news_bot` deploys to a local port (`PUBLIC_PORT`, default `18085`) and does not bind host `:80`.

This keeps it compatible with a shared host where another bot (`astro_mind_bot`) already uses the same nginx instance.

Host nginx should route a dedicated webhook path to this local port, for example:

```nginx
location = /tg/mx/webhook {
    proxy_pass http://127.0.0.1:18085;
}
```

Required GitHub Actions variables/secrets:

- `vars.WEBHOOK_PATH` (example: `/tg/mx/webhook`)
- `vars.ENABLE_HOST_GATEWAY=true` (recommended when DB is on host port, e.g. `localhost:6444` from host perspective)
- `secrets.DEPLOY_CONFIG_JSON` (runtime `config.json`)
- `secrets.AWS_SSH_HOST`, `secrets.AWS_SSH_USER`, `secrets.AWS_SSH_PRIVATE_KEY`
- `secrets.GHCR_USERNAME`, `secrets.GHCR_TOKEN` (for private GHCR pulls)

If app runs in Docker and DB is currently exposed on host `:6444`, set DB host in deploy config to `host.docker.internal`.
