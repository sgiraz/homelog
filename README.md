# HomeLog - Home Expense & Utilities Management

> **Self-hosted, multi-user home expense tracking and utilities management system**

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL%203.0-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js)](https://vuejs.org/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://www.docker.com/)

---

## Table of Contents

- [Features](#features)
- [Tech Stack](#tech-stack)
- [Quick Start](#quick-start)
- [Project Structure](#project-structure)
- [API Documentation](#api-documentation)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

---

## Features

### Core Functionality
- **Multi-Property Support** - Manage multiple properties (current & historical)
- **Expense Tracking** - Categorized expenses with custom categories
- **Expense Splitting** - Split expenses between household members with balance tracking
- **Settlement Tracking** - Record payments between users to settle debts
- **Utilities Management** - Track electricity, gas, water, waste with readings & bills
- **PDF Bill Templates** - Automatic data extraction from utility bills via drag-and-drop template wizard
- **Projects** - Budget tracking for renovations, trips, events
- **Multi-User** - Family support with admin/user roles
- **Dashboard** - Overview with charts (bar, pie, line) and monthly trends

### Advanced Features
- **Analytics** - Interactive charts, consumption trends, cost analysis
- **Bill Management** - PDF upload, automatic field extraction, payment tracking
- **Meter Readings** - Manual readings with comparison (autolettura vs fornitore)
- **Reading Comparison** - Compare self-readings with supplier readings
- **User & Household Settings** - Per-property configuration and split mode settings

---

## Tech Stack

### Backend
- **Language**: Go 1.21+
- **Framework**: Gin (HTTP router)
- **Database**: SQLite (lightweight, embedded, WAL mode)
- **Auth**: JWT tokens (access + refresh)
- **ORM**: GORM
- **PDF Processing**: pdftotext (poppler-utils)

### Frontend
- **Framework**: Vue 3 (Composition API)
- **Build Tool**: Vite
- **State**: Pinia
- **Router**: Vue Router
- **UI**: Tailwind CSS (Apple HIG theme)
- **Charts**: Chart.js
- **Icons**: Lucide Vue

### Infrastructure
- **Containerization**: Docker + Docker Compose
- **Deployment**: Raspberry Pi 3B+ optimized (256MB backend, 128MB frontend)
- **Reverse Proxy**: Nginx (optional)
- **Network**: Tailscale for remote access

---

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 20+
- poppler-utils (for PDF text extraction)
- Docker & Docker Compose (for production)

### Development Setup

```bash
# Clone the repository
git clone https://github.com/sgiraz/homelog.git
cd homelog

# Backend setup
cd backend
go mod download
go run cmd/api/main.go

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
# Frontend: http://your-raspberry-pi-ip:3000
# Backend API: http://your-raspberry-pi-ip:8080
```

---

## Project Structure

```
homelog/
├── backend/                    # Go backend
│   ├── cmd/api/main.go         # Entry point, routes, middleware init
│   ├── internal/
│   │   ├── models/models.go    # GORM models (User, Property, Expense, Utility, etc.)
│   │   ├── handlers/           # HTTP handlers (11 files)
│   │   │   ├── auth.go         # Register, login, refresh token
│   │   │   ├── expense.go      # CRUD + stats
│   │   │   ├── property.go     # CRUD properties
│   │   │   ├── category.go     # CRUD categories
│   │   │   ├── utility.go      # CRUD + readings + bills + comparison
│   │   │   ├── pdf.go          # PDF upload, extraction, templates
│   │   │   ├── project.go      # CRUD projects
│   │   │   ├── settings.go     # User + household settings
│   │   │   ├── balance.go      # Balance calculation between members
│   │   │   ├── settlement.go   # Settlement CRUD
│   │   │   └── member.go       # Household members CRUD
│   │   ├── middleware/         # CORS, JWT auth, rate limiting, logging
│   │   └── database/          # SQLite init, migrations, seeding
│   ├── go.mod
│   └── Dockerfile
│
├── frontend/                   # Vue 3 frontend
│   ├── src/
│   │   ├── main.js             # App entry (Pinia + Router)
│   │   ├── App.vue             # Root component
│   │   ├── router/index.js     # Vue Router with auth guards
│   │   ├── api/client.js       # Axios client with JWT interceptors
│   │   ├── stores/             # Pinia stores
│   │   │   ├── auth.js         # Auth state (login, user, token)
│   │   │   ├── expenses.js     # Expenses state
│   │   │   ├── balance.js      # Balance & settlements state
│   │   │   ├── utilities.js    # Utilities state
│   │   │   └── settings.js     # User settings state
│   │   ├── views/              # Page views
│   │   │   ├── LoginView.vue
│   │   │   ├── DashboardView.vue
│   │   │   ├── ExpensesView.vue
│   │   │   ├── BalanceView.vue
│   │   │   ├── UtilitiesView.vue
│   │   │   └── SettingsView.vue
│   │   ├── components/         # Reusable components (18 files)
│   │   │   ├── common/         # Button, Card, Input
│   │   │   ├── charts/         # BarChart, PieChart, LineChart
│   │   │   ├── layout/         # Navbar
│   │   │   ├── expenses/       # AddExpenseModal, EditExpenseModal
│   │   │   ├── balance/        # SettlementModal
│   │   │   └── utilities/      # TemplateWizard, TemplatesManager, PDFTextractView, etc.
│   │   └── utils/              # Utilities
│   │       ├── tokenizer.js    # PDF text tokenizer
│   │       ├── patternGenerator.js  # Regex pattern generator
│   │       └── dateFormatter.js     # Italian date formatting
│   ├── package.json
│   ├── vite.config.js
│   └── Dockerfile
│
├── data/                       # SQLite database & uploads
├── prototypes/                 # React reference prototypes (3 files)
├── docs/                       # Documentation
│   ├── DEVELOPMENT-GUIDE.md
│   └── SPLIT-SETTLEMENT-SPEC.md
├── docker-compose.yml          # Production deployment
├── docker-compose.dev.yml      # Development deployment
├── .env.example
├── LICENSE
└── README.md
```

---

## API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication
```http
POST /auth/register             # Register new user
POST /auth/login                # Login, returns JWT tokens
POST /auth/refresh              # Refresh access token
```

### Properties
```http
GET    /properties              # List user properties
POST   /properties              # Create property
GET    /properties/:id          # Get property
PUT    /properties/:id          # Update property
DELETE /properties/:id          # Delete property
GET    /properties/:id/balance          # Get balance for property
GET    /properties/:id/balance/details  # Get detailed balance
GET    /properties/:id/settings         # Get household settings
PUT    /properties/:id/settings         # Update household settings
GET    /properties/:id/members          # List household members
POST   /properties/:id/members          # Add household member
```

### Categories
```http
GET    /categories              # List categories
POST   /categories              # Create category
GET    /categories/:id          # Get category
PUT    /categories/:id          # Update category
DELETE /categories/:id          # Delete category
```

### Expenses
```http
GET    /expenses                # List expenses (filterable)
POST   /expenses                # Create expense (supports split)
GET    /expenses/:id            # Get expense
PUT    /expenses/:id            # Update expense
DELETE /expenses/:id            # Delete expense
GET    /expenses/stats          # Get expense statistics
```

### Utilities
```http
GET    /utilities               # List utilities
POST   /utilities               # Create utility
GET    /utilities/:id           # Get utility details
PUT    /utilities/:id           # Update utility
DELETE /utilities/:id           # Delete utility

# Meter readings
POST   /utilities/:id/readings          # Add reading
GET    /utilities/:id/readings          # List readings
PUT    /utilities/:id/readings/:rid     # Update reading
DELETE /utilities/:id/readings/:rid     # Delete reading

# Bills
POST   /utilities/:id/bills             # Add bill
GET    /utilities/:id/bills             # List bills
PUT    /utilities/:id/bills/:bid        # Update bill
PUT    /utilities/:id/bills/:bid/full   # Full bill update
DELETE /utilities/:id/bills/:bid        # Delete bill
POST   /utilities/:id/bills/upload      # Upload bill PDF

# Comparison & contracts
GET    /utilities/:id/compare-readings  # Compare self vs supplier readings
POST   /utilities/contract/upload       # Upload contract PDF
```

### Bill Templates
```http
GET    /templates/bills         # List bill extraction templates
POST   /templates/bills         # Create template
PUT    /templates/bills/:id     # Update template
DELETE /templates/bills/:id     # Delete template
```

### PDF Processing
```http
POST   /pdf/extract-text        # Extract raw text from PDF
POST   /pdf/analyze             # Analyze PDF for template creation
DELETE /pdf/cleanup/:timestamp  # Cleanup temporary template images
```

### Projects
```http
GET    /projects                # List projects
POST   /projects                # Create project
GET    /projects/:id            # Get project
PUT    /projects/:id            # Update project
DELETE /projects/:id            # Delete project
```

### Settings
```http
GET    /settings                # Get user settings
PUT    /settings                # Update user settings
```

### Members
```http
GET    /members/:id             # Get member
PUT    /members/:id             # Update member
DELETE /members/:id             # Delete member
```

### Settlements
```http
GET    /settlements             # List settlements
POST   /settlements             # Create settlement
GET    /settlements/:id         # Get settlement
DELETE /settlements/:id         # Delete settlement
```

---

## Roadmap

### Phase 1: MVP
- [x] Project structure & Docker deployment
- [x] Backend API (Auth, Properties, Categories, Expenses)
- [x] Frontend core views (Login, Dashboard, Expenses)
- [x] Basic charts & analytics (Bar, Pie, Line)

### Phase 2: Utilities & Bills
- [x] Meter readings management
- [x] Bill tracking & PDF upload
- [x] PDF bill template system (drag-and-drop extraction)
- [x] Reading comparison (autolettura vs fornitore)
- [ ] Automatic alerts & reminders
- [ ] Consumption analytics & anomaly detection

### Phase 3: Split & Settlement
- [x] Expense splitting between household members
- [x] Balance calculation
- [x] Settlement tracking
- [x] Household settings per property

### Phase 4: Advanced Features
- [ ] Projects with budget tracking
- [ ] Budget system with alerts
- [ ] Import/Export CSV/Excel
- [ ] PDF report generation
- [ ] Email notifications

### Phase 5: Polish & Community
- [ ] PWA with offline support
- [ ] Dark mode
- [ ] Multi-language (i18n)
- [ ] OCR bill parsing
- [ ] Community templates

---

## Contributing

Contributions are welcome! Please read our [Contributing Guidelines](CONTRIBUTING.md) first.

### Development Workflow
1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style
- **Go**: Follow [Effective Go](https://golang.org/doc/effective_go.html), use `gofmt`
- **Vue**: Use [Vue Style Guide](https://vuejs.org/style-guide/), Composition API with `<script setup>`
- **Commits**: Use [Conventional Commits](https://www.conventionalcommits.org/)

---

## License

This project is licensed under the **AGPL-3.0 License** - see the [LICENSE](LICENSE) file for details.

---

## Support

- **Issues**: [GitHub Issues](https://github.com/sgiraz/homelog/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sgiraz/homelog/discussions)
