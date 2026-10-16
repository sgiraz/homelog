# Self-Hosting

::: warning Work in progress
This guide is being written. For the full, up-to-date deployment reference see
the [DEPLOY-GUIDE.md](https://github.com/sgiraz/homelog/blob/main/DEPLOY-GUIDE.md)
in the repository.
:::

HomeLog ships as a **single Docker container**: the Go binary serves the API and
the embedded frontend, backed by SQLite. No separate database or web server to
run.

## Quick start with Docker

```bash
git clone https://github.com/sgiraz/homelog.git && cd homelog
cp .env.example .env

# Set a secure JWT secret (the only required setting):
#   Linux/macOS:  openssl rand -base64 32
# Paste the output into .env as JWT_SECRET=...

docker compose up -d
```

Then open **http://localhost:8080**, register the first account, and start
tracking.

## Using the pre-built image

```bash
mkdir homelog && cd homelog
echo "JWT_SECRET=$(openssl rand -base64 32)" > .env
curl -O https://raw.githubusercontent.com/sgiraz/homelog/main/docker-compose.yml
docker compose up -d
```

Multi-arch images (amd64 / arm64 / arm/v7) are published on
[Docker Hub](https://hub.docker.com/u/sgira), so the same compose file works on a
Raspberry Pi.

## Choosing a version

The compose file runs `sgira/homelog:${HOMELOG_TAG:-latest}`, so by default you
get the current release and nothing to configure. Set `HOMELOG_TAG` in your
`.env` to run something else:

```bash
HOMELOG_TAG=2026.10.13   # pin an exact version
HOMELOG_TAG=rc           # the latest release candidate
```

Then `docker compose pull && docker compose up -d`.

A release candidate is a build we publish ahead of a release so it can be tried
on a real install. It is not announced as an update and your instance will never
offer one. Running one is a choice with a catch worth knowing: the database
schema migrates on startup and does not migrate back, so going from `rc` to
`latest` afterwards may not start. **Back up `./data` before you switch**, and
stay on `latest` unless you intend to help test.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `JWT_SECRET` | _(required)_ | Secret used to sign auth tokens. |
| `DB_PATH` | `./data/homelog.db` | SQLite database location. |
| `UPLOAD_QUOTA_MB` | `500` | Maximum size of the bill PDFs stored per household, in MB. `0` removes the limit. |
| `PORT` | `8080` | Port the server listens on. |
| `GIN_MODE` | `debug` | Set to `release` in production. |
| `TZ` | _(system)_ | Timezone of the container, e.g. `Europe/Rome`. |
| `CORS_ALLOWED_ORIGINS` | _(localhost only)_ | Comma-separated origins allowed to call the API from another site. Rarely needed: the frontend is served by the same binary. |

Bill PDF uploads are also limited to 20 per user per minute.

## Going further

The repository's
[DEPLOY-GUIDE.md](https://github.com/sgiraz/homelog/blob/main/DEPLOY-GUIDE.md)
covers Raspberry Pi setup, remote access with Tailscale, backups, and hardening.
