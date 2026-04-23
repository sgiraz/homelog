# HomeLog — Deploy Guide

Step-by-step guide for deploying HomeLog on a Raspberry Pi 3B+ with Docker Compose.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Raspberry Pi OS Setup](#2-raspberry-pi-os-setup)
3. [Docker Installation](#3-docker-installation)
4. [Clone & Configure](#4-clone--configure)
5. [Build & Start Services](#5-build--start-services)
   - [5A. Build directly on the Pi](#5a-build-directly-on-the-pi-simple-but-slow)
   - [5B. Pre-build on your PC via DockerHub (faster)](#5b-pre-build-on-your-pc-via-dockerhub-recommended)
6. [Verify Deployment](#6-verify-deployment)
7. [Auto-Start on Boot](#7-auto-start-on-boot)
8. [Backup & Restore](#8-backup--restore)
9. [Security Hardening](#9-security-hardening)
10. [Monitoring & Maintenance](#10-monitoring--maintenance)
11. [Troubleshooting](#11-troubleshooting)

---

## 1. Prerequisites

**Hardware:**
- Raspberry Pi 3B+ (1GB RAM minimum; 2GB+ recommended)
- MicroSD card 16GB+ (Class 10 / A1 or better)
- Power supply 5V 2.5A+

**Software:**
- Raspberry Pi OS (64-bit Lite recommended for headless)
- Internet connection for initial setup

---

## 2. Raspberry Pi OS Setup

### Write OS Image

Use [Raspberry Pi Imager](https://www.raspberrypi.com/software/) on your computer:

1. Select **Raspberry Pi OS Lite (64-bit)**
2. Click the settings icon → enable SSH, set username/password, configure Wi-Fi
3. Write to SD card

### First Boot

```bash
# Find Pi on network
ping raspberrypi.local
# or check your router's DHCP leases for the Pi's IP

# SSH into Pi
ssh pi@raspberrypi.local

# Update system
sudo apt update && sudo apt upgrade -y

# Set hostname (optional)
sudo hostnamectl set-hostname homelog

# Reboot
sudo reboot
```

### Set Static IP (Recommended)

```bash
sudo nano /etc/dhcpcd.conf
```

Add at the end:
```
interface eth0
static ip_address=192.168.1.100/24
static routers=192.168.1.1
static domain_name_servers=192.168.1.1
```

Restart networking:
```bash
sudo systemctl restart dhcpcd
```

---

## 3. Docker Installation

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add your user to docker group (no sudo needed for docker commands)
sudo usermod -aG docker $USER

# Apply group change (or log out and back in)
newgrp docker

# Verify
docker --version
docker compose version
```

Expected output:
```
Docker version 25.x.x, build ...
Docker Compose version v2.x.x
```

---

## 4. Clone & Configure

```bash
# Clone repository
cd ~
git clone https://github.com/sgiraz/homelog.git
cd homelog

# Create data directories
mkdir -p data/uploads

# Copy and edit environment configuration
cp .env.example .env
nano .env
```

### Required .env Settings

```bash
# Generate a secure JWT secret
openssl rand -base64 32
# Copy the output into JWT_SECRET below

# --- REQUIRED ---
JWT_SECRET=PASTE_YOUR_GENERATED_SECRET_HERE

# --- OPTIONAL ---
GIN_MODE=release
TZ=Europe/Rome
```

**IMPORTANT:** Never commit `.env` to version control. It contains secrets.

---

## 5. Build & Start Services

You have two options: build on the Pi directly (simple but slow), or cross-compile on your PC and push to DockerHub (faster).

---

### 5A. Build directly on the Pi (simple but slow)

Building on Raspberry Pi 3B+ takes ~10–15 minutes the first time (Go compilation + frontend build).

```bash
docker compose build
docker compose up -d
```

Expected output (truncated):
```
[+] Building ...
 => [frontend-builder] FROM node:24-alpine
 => [frontend-builder] npm run build ...
 => [backend-builder] FROM golang:1.25-alpine
 => [backend-builder] go build -ldflags="-s -w" -o homelog ...
```

---

### 5B. Pre-build on your PC via DockerHub (recommended)

HomeLog ships as a **single container**: the Go binary embeds the compiled frontend via `go:embed`. Cross-compile it on your PC and push to DockerHub — the Pi only needs to pull and run, no compiler needed.

#### On your development machine

**Prerequisites:** Docker with buildx support (Docker Desktop includes this).

```bash
# Log in to DockerHub
docker login
# (enter your DockerHub username and password)

# Create a multi-platform builder (one-time setup)
docker buildx create --name mybuilder --use
docker buildx inspect --bootstrap

# Build and push for ARM64 (Raspberry Pi 3B+ with 64-bit OS)
# Replace YOUR_DOCKERHUB_USERNAME with your DockerHub account
docker buildx build \
  --platform linux/arm64 \
  -t YOUR_DOCKERHUB_USERNAME/homelog:latest \
  --push \
  .
```

#### On the Raspberry Pi

Edit `docker-compose.yml` to pull the pre-built image instead of building locally:

```bash
# Replace the build: block with image: in docker-compose.yml
# Change:
#   build:
#     context: .
#     dockerfile: Dockerfile
# To:
#   image: YOUR_DOCKERHUB_USERNAME/homelog:latest

docker compose pull
docker compose up -d
```

#### Updating to a new version

On your dev machine: rebuild and push with the same command (or add a version tag like `:v1.1`).
On the Pi:

```bash
docker compose pull
docker compose up -d
```

---

### Check Status

```bash
# Check container is running and healthy (wait ~40s for health checks to pass)
docker ps

# Expected output:
# homelog    Up X minutes (healthy)
```

---

## 6. Verify Deployment

```bash
# Health check (Go binary serves API + embedded frontend on the same port)
curl http://localhost:8080/health
# Expected: {"service":"homelog-api","status":"healthy","uptime":"...","version":"1.0.0"}

# Open in browser
# http://192.168.1.100:8080
```

### First Login

1. Navigate to `http://<pi-ip>:8080`
2. Click **Register** to create the first admin account
3. Add your first property in Settings
4. Start tracking expenses and utilities

---

## 7. Auto-Start on Boot

Install the systemd service so HomeLog starts automatically after a reboot:

```bash
# Install service file
sudo cp scripts/homelog.service /etc/systemd/system/

# If your user is not 'pi', edit the WorkingDirectory in the service file:
# sudo nano /etc/systemd/system/homelog.service
# Change: WorkingDirectory=/home/pi/homelog
# To:     WorkingDirectory=/home/YOUR_USERNAME/homelog

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable homelog
sudo systemctl start homelog

# Verify
sudo systemctl status homelog
```

Test by rebooting:
```bash
sudo reboot
# After reboot, wait ~2 minutes then check:
curl http://localhost:8080/health
```

---

## 8. Backup & Restore

### Manual Backup

```bash
# Copy database file from running container
docker compose exec -T homelog cp /app/data/homelog.db /app/data/backup-$(date +%Y%m%d).db

# Or copy from host (data/ directory is a mounted volume)
cp ~/homelog/data/homelog.db ~/homelog/data/backup-$(date +%Y%m%d).db
```

### Automated Backup via Cron

```bash
crontab -e
```

Add (daily backup at 2 AM, keep 30 days):
```cron
0 2 * * * cp ~/homelog/data/homelog.db ~/homelog/data/backup-$(date +\%Y\%m\%d).db
0 3 * * * find ~/homelog/data -name 'backup-*.db' -mtime +30 -delete
```

### Export via UI

Settings → Backup & Dati → **Esporta Tutto** downloads a JSON file with all your data.

### Restore from Backup

```bash
# Stop services
docker compose down

# Replace database (use a dated backup or the exported JSON)
cp ~/homelog/data/backup-YYYYMMDD.db ~/homelog/data/homelog.db

# Remove WAL files if present (they belong to the old DB session)
rm -f ~/homelog/data/homelog.db-wal ~/homelog/data/homelog.db-shm

# Restart
docker compose up -d
```

Or restore from a JSON export via the UI: Settings → Backup & Dati → **Importa**.

---

## 9. Security Hardening

### Change Default JWT Secret

```bash
# Generate strong secret
openssl rand -base64 32

# Update .env
nano ~/homelog/.env
# JWT_SECRET=<your-generated-secret>

# Restart backend
docker compose restart backend
```

### Enable UFW Firewall

```bash
sudo apt install ufw

# Default policies
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH (CRITICAL - do this first or you'll be locked out)
sudo ufw allow 22

# Allow HomeLog (restrict to LAN if preferred)
sudo ufw allow 8080   # API + embedded frontend

# Enable
sudo ufw enable
sudo ufw status
```

To restrict to LAN only (e.g., 192.168.1.0/24):
```bash
sudo ufw allow from 192.168.1.0/24 to any port 8080
sudo ufw delete allow 8080
```

### Keep System Updated

```bash
sudo apt update && sudo apt upgrade -y
```

Set up automatic security updates:
```bash
sudo apt install unattended-upgrades
sudo dpkg-reconfigure unattended-upgrades
```

---

## 10. Monitoring & Maintenance

### Resource Usage

```bash
# Docker container stats (live)
docker stats

# One-time snapshot
docker stats --no-stream

# System memory
free -h

# Disk usage
df -h
du -sh ~/homelog/data/
```

Expected resource usage (Raspberry Pi 3B+):
| Service  | CPU   | Memory |
|----------|-------|--------|
| homelog  | <5%   | ~80MB  |

### Logs

```bash
# All services
docker compose logs -f

# Last 100 lines
docker compose logs --tail=100 homelog

# Log files are stored in Docker's json-file driver
# Max size: 10MB per file, 3 files (configured in docker-compose.yml)
```

### Update HomeLog

```bash
cd ~/homelog

# Pull latest code
git pull

# Rebuild and restart
docker compose build
docker compose up -d

# Verify
docker ps
curl http://localhost:8080/health
```

### Increase Swap (if needed)

If builds fail or containers OOM on a 1GB Pi:

```bash
sudo dphys-swapfile swapoff
sudo nano /etc/dphys-swapfile
# Change: CONF_SWAPSIZE=1024
sudo dphys-swapfile setup
sudo dphys-swapfile swapon
free -h
```

---

## 11. Troubleshooting

### Container won't start

```bash
docker compose logs homelog
```

### Backend: database locked

Symptoms: `database is locked` errors in backend logs.

```bash
docker compose down
rm -f ~/homelog/data/homelog.db-wal ~/homelog/data/homelog.db-shm
docker compose up -d
```

### Health check failing

```bash
# Manual check
curl -v http://localhost:8080/health

# Container inspect
docker inspect homelog | grep -A 20 Health
```

### Port already in use

```bash
# Find what's using port 8080
sudo ss -tlnp | grep 8080

# Kill conflicting process or change the port in docker-compose.yml
```

### Out of disk space

```bash
df -h

# Clean old Docker images
docker image prune -f

# Clean old backup files
ls -lh ~/homelog/data/backup-*.db
rm ~/homelog/data/backup-YYYYMMDD.db

# Clean Docker logs
docker compose down
docker compose up -d
```

### Cannot SSH after enabling UFW

If locked out, physically connect keyboard + monitor and run:

```bash
sudo ufw allow 22
sudo ufw status
```

---

## Pre-Deploy Checklist

- [ ] JWT_SECRET changed (not the default)
- [ ] Static IP configured on Pi
- [ ] UFW firewall enabled
- [ ] Systemd service enabled for auto-start
- [ ] Automated backup cron configured
- [ ] Swap increased if Pi has 1GB RAM
- [ ] First user account created and tested

---

*HomeLog v1.0.0 — AGPL-3.0*
