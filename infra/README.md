# infra

Operational glue for the shared droplet: caddy reverse proxy, ellie-ai, and
resto-demo.

Core services (api, frontend, mysql, redis) and the dev data layer have moved to
`sf-voice/core` — see `core/infra/`.

## layout

- `deploy/compose.prod.yml` — caddy + ellie + resto production compose stack.
- `deploy/sfctl.sh` — production control script, installed on the droplet as
  `/srv/sf-voice/bin/sfctl`.
- `deploy/sfctl.d/` — modular sfctl commands (deploy, preview teardown).
- `deploy/Caddyfile` — reverse proxy config for all services.
- `deploy/smoke-vad.py` — deploy smoke test for ellie's VAD websocket.

## deploy console

The `deploy console` workflow handles caddy, ellie, and resto. Manual examples:

```text
operation=status service=all
operation=deploy service=caddy tag=latest
operation=deploy service=ellie tag=sha-<sha>
operation=logs service=ellie log_lines=300
operation=rollback service=resto tag=sha-<previous-sha>
```

On the droplet:

```bash
/srv/sf-voice/bin/sfctl status all
/srv/sf-voice/bin/sfctl deploy caddy latest
/srv/sf-voice/bin/sfctl deploy ellie sha-<sha>
/srv/sf-voice/bin/sfctl logs ellie 300
```

## secrets

| GH secret               | used by      |
| ----------------------- | ------------ |
| `SECRET_KEY_BASE`       | resto, ellie |
| `INTERNAL_API_TOKEN`    | resto, ellie |
| `OPENAI_API_KEY`        | ellie        |
| `TELNYX_API_KEY`        | ellie        |
| `TELNYX_PUBLIC_KEY`     | ellie        |
| `PHONE_NUMBER`          | ellie        |
| `STAFF_PHONE_E164`      | ellie        |
| `AWS_ACCESS_KEY_ID`     | ellie        |
| `AWS_SECRET_ACCESS_KEY` | ellie        |
| `AWS_REGION`            | ellie        |
| `S3_BUCKET_NAME`        | ellie        |
| `DROPLET_HOST`          | runner       |
| `DROPLET_SSH_KEY`       | runner       |

Core API secrets (`DATABASE_URL`, `REDIS_URL`, `CLICKHOUSE_*`, `QDRANT_*`,
`AUTUMN_SECRET_KEY`, etc.) are managed in the `sf-voice/core` repo and are not
present here.

## runtime services

| service      | image                                     | data                   | public host              |
| ------------ | ----------------------------------------- | ---------------------- | ------------------------ |
| `caddy`      | `caddy:2-alpine`                          | caddy volumes + certs  | public `80/443`          |
| `ellie-ai`   | `ghcr.io/sf-voice/ellie-ai`               | sqlite in `data/ellie` | `ellie-ai.sf-voice.sh`   |
| `resto-demo` | `ghcr.io/sf-voice/restaurant-booking-app` | sqlite in `data/resto` | `resto-demo.sf-voice.sh` |
