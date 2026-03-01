# HomeLog

> Self-hosted home expense tracking and utilities management for families.

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL%203.0-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js)](https://vuejs.org/)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker)](https://hub.docker.com/u/sgira)

<!-- TODO: Add screenshot here -->
<!-- ![HomeLog Dashboard](docs/images/dashboard.png) -->

---

## Features

- **Expense Tracking** with categories, subcategories, and project-based budgets
- **Expense Splitting** between household members with automatic balance and settlement tracking
- **Utilities Management** for electricity, gas, water — meter readings, bills, consumption analysis
- **PDF Bill Templates** with drag-and-drop field extraction wizard
- **Interactive Dashboard** with adaptive trend charts (daily/monthly/quarterly granularity)
- **Projects** for budget tracking (renovations, trips, events)
- **Multi-User / Multi-Property** support
- **Mobile-First UX** with bottom navigation, bottom sheet modals, and touch-friendly design
- **Export/Import** your data as JSON
- **Runs anywhere** — Raspberry Pi, VPS, NAS, or any Docker host

---

## Quick Start

### Docker (recommended)

```bash
git clone https://github.com/sgiraz/homelog.git && cd homelog
cp .env.example .env

# Set a secure JWT secret (the only required setting)
# Linux/macOS: openssl rand -base64 32
# Paste the output into .env as JWT_SECRET=...

docker compose up -d
```

Open **http://localhost:3000**, register, and start tracking.

### Pre-built images from DockerHub

```bash
mkdir homelog && cd homelog

# Create .env with your JWT secret
echo "JWT_SECRET=$(openssl rand -base64 32)" > .env

# Download and run
curl -O https://raw.githubusercontent.com/sgiraz/homelog/main/docker-compose.yml
docker compose up -d
```

See [DEPLOY-GUIDE.md](DEPLOY-GUIDE.md) for Raspberry Pi setup, Tailscale remote access, backups, and more.

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go, Gin, GORM, SQLite (WAL mode) |
| Frontend | Vue 3, Vite, Pinia, Tailwind CSS, Chart.js |
| Auth | JWT (access + refresh tokens) |
| PDF | pdftotext (poppler-utils) |
| Deploy | Docker Compose, Nginx reverse proxy, multi-arch images (amd64/arm64/arm/v7) |

---

## Documentation

| Document | Description |
|----------|-------------|
| [DEPLOY-GUIDE.md](DEPLOY-GUIDE.md) | Full deployment guide (Raspberry Pi, Docker, Tailscale, backups, security) |
| [docs/API.md](docs/API.md) | REST API reference |
| [docs/SPLIT-SETTLEMENT-SPEC.md](docs/SPLIT-SETTLEMENT-SPEC.md) | Split & settlement system design |

---

## Development

```bash
# Backend (terminal 1)
cd backend && go mod download && go run cmd/api/main.go

# Frontend (terminal 2)
cd frontend && npm install && npm run dev
```

Frontend: http://localhost:5173 — Backend API: http://localhost:8080

The Vite dev server proxies `/api` to the backend automatically.

---

## Contributing

Contributions are welcome! Here's how:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit with [Conventional Commits](https://www.conventionalcommits.org/) (`git commit -m 'feat: add my feature'`)
4. Push and open a Pull Request

**Code style:** Go (`gofmt`), Vue (`<script setup>` Composition API), Tailwind CSS.

---

## Support the Project

If HomeLog is useful to you, consider supporting its development:

- Star this repository
- [Report bugs or suggest features](https://github.com/sgiraz/homelog/issues)
- [Join the discussion](https://github.com/sgiraz/homelog/discussions)
- Contribute code, translations, or documentation
<!-- - [Sponsor on GitHub](https://github.com/sponsors/sgiraz) -->

---

## License

[AGPL-3.0](LICENSE) — free to use, modify, and self-host. If you distribute a modified version, you must share the source code.
