# 📁 Struttura Progetto HomeLog

```
homelog/
├── 📄 README.md                        ✅ Overview progetto
├── 📄 README_FOR_CLAUDE_CODE.md        ✅ Guida completa per Claude Code
├── 📄 QUICKSTART.md                    ✅ Quick start
├── 📄 STRUCTURE.md                     ✅ Questo file
├── 📄 docker-compose.yml               ✅ Docker orchestration
├── 📄 .env.example                     ✅ Environment variables template
│
├── 📁 backend/                         Backend Go
│   ├── 📄 Dockerfile                   ✅ Multi-stage ARM build
│   ├── 📄 go.mod                       ✅ Go dependencies
│   ├── 📁 cmd/
│   │   └── api/
│   │       └── 📄 main.go              ✅ Server entry point (COMPLETO)
│   └── 📁 internal/
│       ├── 📁 models/
│       │   └── 📄 models.go            ✅ Database models (11 modelli)
│       ├── 📁 database/
│       │   └── 📄 database.go          ✅ DB init + migrations + seed
│       ├── 📁 middleware/
│       │   └── 📄 middleware.go        ✅ CORS + JWT + Rate Limit
│       └── 📁 handlers/
│           ├── 📄 auth.go              ✅ Auth handler (ESEMPIO COMPLETO)
│           ├── 📄 expense.go           ⏳ DA COMPLETARE (template + TODOs)
│           ├── 📄 property.go          ⏳ DA COMPLETARE (stub)
│           ├── 📄 category.go          ⏳ DA COMPLETARE (stub)
│           ├── 📄 utility.go           ⏳ DA COMPLETARE (stub)
│           ├── 📄 project.go           ⏳ DA COMPLETARE (stub)
│           ├── 📄 settings.go          ⏳ DA COMPLETARE (stub)
│           ├── 📄 balance.go           ⏳ DA COMPLETARE (template + TODOs)
│           └── 📄 settlement.go        ⏳ DA COMPLETARE (stub)
│
├── 📁 frontend/                        Frontend Vue 3
│   ├── 📄 package.json                 ✅ Dependencies Vue
│   ├── 📄 vite.config.js               ✅ Vite + PWA config
│   ├── 📄 tailwind.config.js           ✅ Tailwind theme (Apple HIG)
│   ├── 📄 index.html                   ✅ Entry HTML
│   ├── 📄 postcss.config.js            ✅ PostCSS config
│   └── 📁 src/
│       ├── 📄 main.js                  ✅ Vue entry point
│       ├── 📄 App.vue                  ✅ Root component (base)
│       ├── 📁 router/
│       │   └── 📄 index.js             ✅ Vue Router (completo)
│       ├── 📁 api/
│       │   └── 📄 client.js            ✅ Axios client (completo)
│       ├── 📁 stores/                  ⏳ DA CREARE (Pinia stores)
│       ├── 📁 views/                   ⏳ DA CREARE (tutte le views)
│       ├── 📁 components/              ⏳ DA CREARE (tutti i components)
│       └── 📁 assets/
│           └── styles/
│               └── 📄 main.css         ✅ Tailwind imports
│
├── 📁 docs/                            Documentazione
│   ├── 📄 DEVELOPMENT-GUIDE.md         ✅ Guide complete sviluppo
│   └── 📄 SPLIT-SETTLEMENT-SPEC.md     ✅ Spec feature split/settlement
│
├── 📁 prototypes/                      Prototipi React (riferimento UI)
│   ├── 📄 homelog-prototype-v3-FINAL.jsx
│   ├── 📄 homelog-with-balance.jsx
│   └── 📄 homelog-split-prototype.jsx
│
└── 📁 data/                            Database + uploads (creato al runtime)
    ├── homelog.db                      SQLite database
    └── uploads/                        File uploads (PDF bollette, etc.)

Legenda:
✅ = Completato e funzionante
⏳ = Da implementare (con template/TODOs)
📄 = File
📁 = Directory
```

## 📊 Stato Completamento

### Backend: 40% ✅
- ✅ Struttura progetto
- ✅ Database models completi
- ✅ Auth system completo
- ✅ Middleware completi
- ⏳ Handlers (8 da completare)

### Frontend: 20% ✅
- ✅ Configurazione completa
- ✅ Router
- ✅ API client
- ⏳ Stores (da creare)
- ⏳ Views (da creare)
- ⏳ Components (da creare)

### DevOps: 100% ✅
- ✅ Docker Compose
- ✅ Dockerfile backend
- ✅ Dockerfile frontend
- ✅ .env.example

### Documentazione: 100% ✅
- ✅ README principale
- ✅ README per Claude Code
- ✅ Quick Start Guide
- ✅ Development Guide
- ✅ Split/Settlement Spec

## 🎯 Prossimi Step

1. **Backend Handlers** (Priorità 1)
   - expense.go - Gestione spese
   - balance.go - Calcolo bilancio
   - settlement.go - Gestione pagamenti
   - property.go - Gestione proprietà
   - category.go - Gestione categorie
   - utility.go - Gestione utilities
   - project.go - Gestione progetti
   - settings.go - Impostazioni

2. **Frontend Vue** (Priorità 2)
   - Stores Pinia (auth, expenses, balance, utilities, projects)
   - Views (Dashboard, Expenses, Balance, Utilities, Settings)
   - Components (Card, Button, Modal, Charts, Navbar, Sidebar)
   - Convertire prototipi React in Vue

3. **Testing** (Priorità 3)
   - Backend: test handlers con Postman
   - Frontend: test navigazione e features
   - Integration: test completo frontend-backend
   - E2E: test user flow completo

4. **Deploy** (Priorità 4)
   - Build Docker
   - Deploy su Raspberry Pi
   - Test da mobile (PWA)

## 📚 Risorse

- **Per implementazione**: Leggi `README_FOR_CLAUDE_CODE.md`
- **Per esempi**: Guarda `backend/internal/handlers/auth.go`
- **Per UI**: Converti `prototypes/*.jsx` in Vue
- **Per API**: Consulta `docs/DEVELOPMENT-GUIDE.md`

## 💪 Ready to Code!

Apri il progetto in VS Code e chiedi a Claude Code di implementare seguendo le priorità!
