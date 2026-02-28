# 🚀 HomeLog - Quick Start Guide

**For Claude Code Opus 4.5** and the user (Simone)

---

## 📋 BEFORE YOU START

Read `/README-FOR-CLAUDE-CODE.md` first - it has complete context, architecture, and examples.

---

## 🏁 GETTING STARTED

### 1. Open in VS Code
```bash
cd /path/to/homelog
code .
```

### 2. Open README-FOR-CLAUDE-CODE.md
Claude Code: Read this file first to understand the project.

### 3. Follow Priority Order
Do in this exact order:

**Phase 1: Backend Core** (MOST IMPORTANT)
1. Update `backend/internal/models/models.go` - add split/settlement models
2. Update `backend/internal/database/database.go` - AutoMigrate new models
3. Complete `backend/internal/handlers/expense.go` - implement TODOs
4. Complete `backend/internal/handlers/balance.go` - implement calculation
5. Complete `backend/internal/handlers/settlement.go` - implement creation

**Phase 2: Backend Additional**
6. Complete other handlers (property, category, utility, project, settings)

**Phase 3: Frontend**
7. Create stores (auth, expenses, balance)
8. Create views (Dashboard, Expenses, Balance)
9. Create components (Card, Button, Modal, Charts)
10. Convert React prototype to Vue

---

## 🧪 TESTING

### Backend
```bash
cd backend
go run cmd/api/main.go

# In another terminal:
curl http://localhost:8080/health
```

### Frontend
```bash
cd frontend
npm install
npm run dev
```

---

## 📁 KEY FILES TO START WITH

1. `/README-FOR-CLAUDE-CODE.md` - **READ THIS FIRST**
2. `/backend/internal/handlers/expense.go` - Example handler with TODOs
3. `/backend/internal/handlers/auth.go` - Complete handler as reference
4. `/backend/internal/models/models.go` - Add new models here
5. `/frontend/src/router/index.js` - Already done
6. `/mnt/user-data/outputs/homelog-split-prototype.jsx` - React UI reference

---

## 💻 FOR THE USER (Simone)

### Setup Development Environment
```bash
# 1. Install Go
brew install go  # macOS
# or download from https://go.dev/dl/

# 2. Install Node.js
brew install node  # macOS
# or download from https://nodejs.org/

# 3. Verify installations
go version   # Should be 1.21+
node --version   # Should be 20+
npm --version

# 4. Open project
cd homelog
code .
```

### Ask Claude Code to implement
In VS Code with Claude Code extension:
1. Open README-FOR-CLAUDE-CODE.md
2. Ask Claude Code: "Read README-FOR-CLAUDE-CODE.md and implement Phase 1"
3. Let Claude Code work through the TODOs
4. Test each phase before moving to next

---

## 🎯 WHAT'S ALREADY DONE

✅ Project structure
✅ Database models (base)
✅ Auth system (complete)
✅ Middleware (CORS, JWT, rate limit)
✅ Docker config
✅ Frontend config (Vite, Tailwind)
✅ React prototype (for reference)
✅ Documentation

## ❌ WHAT NEEDS IMPLEMENTATION

❌ Backend handlers (expense, balance, settlement, etc.)
❌ Database updates (split/settlement models)
❌ Frontend Vue app (all views and components)
❌ Convert React prototype to Vue

---

## 📚 USEFUL COMMANDS

```bash
# Backend
cd backend
go mod tidy              # Install dependencies
go run cmd/api/main.go  # Start server
go test ./...           # Run tests

# Frontend
cd frontend
npm install             # Install dependencies
npm run dev            # Start dev server
npm run build          # Build for production

# Docker (when ready)
docker-compose up -d   # Start all services
docker-compose logs -f # View logs
docker-compose down    # Stop all services
```

---

## 🐛 COMMON ISSUES

### "Package not found"
```bash
cd backend
go mod tidy
```

### "Module not found" (frontend)
```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

### Database errors
```bash
# Delete and recreate
rm data/homelog.db
# Restart backend - it will auto-migrate
```

---

## ✅ HOW TO VERIFY IT WORKS

### Backend Working:
```bash
curl http://localhost:8080/health
# Should return: {"status":"ok", ...}

curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"password123","name":"Test"}'
# Should return: {"token":"...", "user":{...}}
```

### Frontend Working:
- Open http://localhost:5173
- Should see login page
- Can login and navigate

---

## 🎓 LEARNING RESOURCES (for later)

When ready to learn Go and Vue:
- Go Tour: https://go.dev/tour/
- Vue Guide: https://vuejs.org/guide/
- GORM Docs: https://gorm.io/docs/
- Gin Docs: https://gin-gonic.com/docs/

---

**Ready? Open VS Code, read README-FOR-CLAUDE-CODE.md, and start coding!** 🚀
