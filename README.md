# 🏠 HomeLog - Home Expense & Utilities Management

> **Self-hosted, multi-user home expense tracking and utilities management system**

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL%203.0-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js)](https://vuejs.org/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://www.docker.com/)

---

## 📋 Table of Contents

- [Features](#-features)
- [Tech Stack](#-tech-stack)
- [Quick Start](#-quick-start)
- [Installation](#-installation)
- [Project Structure](#-project-structure)
- [API Documentation](#-api-documentation)
- [Roadmap](#-roadmap)
- [Contributing](#-contributing)
- [License](#-license)

---

## ✨ Features

### Core Functionality
- 🏠 **Multi-Property Support** - Manage multiple properties (current & historical)
- 💰 **Expense Tracking** - Categorized expenses with custom categories
- 💡 **Utilities Management** - Track electricity, gas, water, waste with readings & bills
- 📊 **Projects** - Budget tracking for renovations, trips, events
- 👥 **Multi-User** - Family support with admin/user roles
- 📅 **Smart Alerts** - Bill due dates, anomaly detection, reading reminders

### Advanced Features
- 📈 **Analytics** - Interactive charts, consumption trends, cost analysis
- 📄 **Bill Management** - PDF attachments, payment tracking, historical data
- 📸 **Meter Readings** - Manual readings with photo upload
- 🔄 **Import/Export** - CSV, Excel, JSON support
- 🌙 **Dark Mode** - Full dark theme support
- 📱 **PWA** - Installable on iOS/Android with offline support

---

## 🛠 Tech Stack

### Backend
- **Language**: Go 1.21+
- **Framework**: Gin (HTTP router)
- **Database**: SQLite (lightweight, embedded)
- **Auth**: JWT tokens
- **ORM**: GORM

### Frontend
- **Framework**: Vue 3 (Composition API)
- **Build Tool**: Vite
- **State**: Pinia
- **Router**: Vue Router
- **UI**: Tailwind CSS
- **Charts**: Recharts / Chart.js
- **Icons**: Lucide Vue

### Infrastructure
- **Containerization**: Docker + Docker Compose
- **Deployment**: Raspberry Pi 3B+ optimized
- **Reverse Proxy**: Nginx (optional)
- **Network**: Tailscale for remote access

---

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Node.js 20+
- Docker & Docker Compose (for production)
- 256MB+ RAM available (Raspberry Pi compatible)

### Development Setup

```bash
# Clone the repository
git clone https://github.com/sgiraz/homelog.git
cd homelog

# Backend setup
cd backend
go mod download
go run main.go

# Frontend setup (new terminal)
cd ../frontend
npm install
npm run dev

# Access the app
# Frontend: http://localhost:5173
# Backend API: http://localhost:8080
```

### Production Deploy (Docker Compose)

```bash
# Edit configuration
cp .env.example .env
nano .env

# Start services
docker-compose up -d

# Access the app
# http://your-raspberry-pi-ip:8080
```

---

## 📁 Project Structure

```
homelog/
├── backend/                # Go backend
│   ├── cmd/
│   │   └── api/           # Main application
│   ├── internal/
│   │   ├── models/        # Database models
│   │   ├── handlers/      # HTTP handlers
│   │   ├── middleware/    # Auth, CORS, etc.
│   │   ├── services/      # Business logic
│   │   └── database/      # DB connection
│   ├── pkg/               # Shared packages
│   ├── migrations/        # SQL migrations
│   ├── go.mod
│   └── Dockerfile
│
├── frontend/              # Vue 3 frontend
│   ├── src/
│   │   ├── components/    # Vue components
│   │   ├── views/         # Page views
│   │   ├── stores/        # Pinia stores
│   │   ├── router/        # Vue Router
│   │   ├── assets/        # Static assets
│   │   └── api/           # API client
│   ├── public/
│   ├── package.json
│   ├── vite.config.js
│   └── Dockerfile
│
├── data/                  # SQLite database & uploads
│   ├── homelog.db
│   └── uploads/
│
├── docker/                # Docker configurations
│   ├── nginx/
│   └── scripts/
│
├── docs/                  # Documentation
│   ├── API.md
│   ├── ARCHITECTURE.md
│   └── DEPLOYMENT.md
│
├── docker-compose.yml
├── .env.example
├── LICENSE
└── README.md
```

---

## 📡 API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication
```http
POST /auth/register
POST /auth/login
POST /auth/refresh
```

### Expenses
```http
GET    /expenses           # List expenses (filterable)
POST   /expenses           # Create expense
GET    /expenses/:id       # Get expense
PUT    /expenses/:id       # Update expense
DELETE /expenses/:id       # Delete expense
```

### Utilities
```http
GET    /utilities          # List utilities
POST   /utilities          # Create utility
GET    /utilities/:id      # Get utility details
POST   /utilities/:id/readings  # Add meter reading
POST   /utilities/:id/bills     # Add bill
```

### Projects
```http
GET    /projects           # List projects
POST   /projects           # Create project
GET    /projects/:id       # Get project
PUT    /projects/:id       # Update project
DELETE /projects/:id       # Delete project
```

Full API documentation: [docs/API.md](docs/API.md)

---

## 🗺 Roadmap

### Phase 1: MVP (Current)
- [x] Project structure
- [x] UI/UX prototype
- [ ] Backend API (Expenses, Properties, Auth)
- [ ] Frontend core views
- [ ] Docker deployment
- [ ] Basic charts & analytics

### Phase 2: Utilities (Next)
- [ ] Meter readings management
- [ ] Bill tracking & PDF upload
- [ ] Automatic alerts
- [ ] Consumption analytics
- [ ] Rate history tracking

### Phase 3: Advanced Features
- [ ] Budget system
- [ ] Project tracking
- [ ] Import/Export CSV/Excel
- [ ] PDF report generation
- [ ] Email notifications

### Phase 4: Community (Future)
- [ ] OCR bill parsing
- [ ] AI anomaly detection
- [ ] Multi-language (i18n)
- [ ] Community templates
- [ ] Plugin system

---

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guidelines](CONTRIBUTING.md) first.

### Development Workflow
1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style
- **Go**: Follow [Effective Go](https://golang.org/doc/effective_go.html)
- **Vue**: Use [Vue Style Guide](https://vuejs.org/style-guide/)
- **Commits**: Use [Conventional Commits](https://www.conventionalcommits.org/)

---

## 📄 License

This project is licensed under the **AGPL-3.0 License** - see the [LICENSE](LICENSE) file for details.

### Why AGPL-3.0?
We chose AGPL to protect the open-source nature of this project, especially for web-based deployments. Any modifications to the code must be shared back with the community.

---

## 🙏 Acknowledgments

- Built with ❤️ for the open-source community
- Inspired by the need for privacy-focused home management
- Optimized for Raspberry Pi self-hosting

---

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/sgiraz/homelog/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sgiraz/homelog/discussions)
- **Email**: support@homelog.app

---

**Made with 🏠 by the HomeLog Team**
