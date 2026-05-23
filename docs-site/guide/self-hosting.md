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

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `JWT_SECRET` | _(required)_ | Secret used to sign auth tokens. |
| `DB_PATH` | `./data/homelog.db` | SQLite database location. |
| `GIN_MODE` | `debug` | Set to `release` in production. |

## Going further

The repository's
[DEPLOY-GUIDE.md](https://github.com/sgiraz/homelog/blob/main/DEPLOY-GUIDE.md)
covers Raspberry Pi setup, remote access with Tailscale, backups, and hardening.
