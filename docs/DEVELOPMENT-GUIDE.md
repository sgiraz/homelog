# 🏠 HomeLog - Progetto Completo: Riepilogo e Guida

## 📦 COSA È STATO CREATO

### ✅ Struttura Progetto Completa
```
homelog/
├── backend/               # Backend Go
│   ├── cmd/api/
│   │   └── main.go       # ✅ Entry point completo
│   ├── internal/
│   │   ├── models/
│   │   │   └── models.go # ✅ Tutti i modelli database
│   │   ├── handlers/
│   │   │   └── auth.go   # ✅ Handler autenticazione
│   │   ├── middleware/
│   │   │   └── middleware.go # ✅ CORS, JWT, Rate Limit
│   │   └── database/
│   │       └── database.go   # ✅ Init DB + migrations
│   ├── go.mod            # ✅ Dipendenze Go
│   └── Dockerfile        # ✅ Multi-stage ottimizzato ARM
│
├── frontend/             # Frontend Vue 3
│   ├── package.json      # ✅ Dipendenze Vue
│   ├── vite.config.js    # ✅ Vite + PWA config
│   ├── tailwind.config.js # ✅ Tailwind + iOS theme
│   └── src/              # ⏭️ DA COMPLETARE (vedi sotto)
│
├── docker-compose.yml    # ✅ Deploy Raspberry Pi
├── .env.example          # ✅ Variabili ambiente
└── README.md             # ✅ Documentazione completa
```

### ✅ Backend Go - File Creati

1. **main.go** - Server completo con:
   - Inizializzazione database
   - Routes API v1
   - Middleware CORS, Auth, Rate Limiting
   - Health check endpoint

2. **models.go** - Tutti i modelli:
   - User, Property, Category, Subcategory
   - Expense, Utility, MeterReading, Bill
   - UtilityRate, Project, UserSettings

3. **database.go** - Database management:
   - Init SQLite con WAL mode
   - Auto-migration
   - Seed default categories

4. **auth.go** - Autenticazione completa:
   - Register (con seed dati default)
   - Login (con password hash bcrypt)
   - Refresh token (JWT)

5. **middleware.go** - Middleware:
   - CORS configurabile
   - JWT validation
   - Admin role check
   - Rate limiting (100 req/min)
   - Logger

### ✅ Frontend Vue - File Creati

1. **package.json** - Dipendenze:
   - Vue 3 + Vue Router + Pinia
   - Chart.js + Lucide Icons
   - Axios per API calls
   - PWA plugin

2. **vite.config.js** - Configurazione:
   - PWA manifest per iOS
   - Service worker con cache
   - Proxy API per development

3. **tailwind.config.js** - Design system:
   - Colori utilities (light giallo, gas arancio, etc.)
   - 8pt grid spacing
   - iOS border radius
   - Liquid glass shadows

### ✅ Docker - File Creati

1. **docker-compose.yml** - Orchestration:
   - Backend (256MB RAM limit)
   - Frontend (128MB RAM limit)
   - Health checks
   - Network isolation

2. **Dockerfile (backend)** - Multi-stage:
   - Build con Go 1.21
   - Runtime Alpine Linux
   - ARM7 per Raspberry Pi
   - Non-root user

---

## ⏭️ COSA MANCA (DA COMPLETARE)

### Backend Handlers (da creare)

**File da creare in `/backend/internal/handlers/`:**

1. **expense.go** - Gestione spese
   ```go
   type ExpenseHandler struct {}
   func (h *ExpenseHandler) List(c *gin.Context) { /* GET /expenses */ }
   func (h *ExpenseHandler) Create(c *gin.Context) { /* POST /expenses */ }
   func (h *ExpenseHandler) Get(c *gin.Context) { /* GET /expenses/:id */ }
   func (h *ExpenseHandler) Update(c *gin.Context) { /* PUT /expenses/:id */ }
   func (h *ExpenseHandler) Delete(c *gin.Context) { /* DELETE /expenses/:id */ }
   func (h *ExpenseHandler) GetStats(c *gin.Context) { /* GET /expenses/stats */ }
   ```

2. **property.go** - Gestione abitazioni
3. **category.go** - Gestione categorie
4. **utility.go** - Gestione utilities
5. **project.go** - Gestione progetti
6. **settings.go** - Gestione impostazioni

**Template per un handler:**
```go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    "github.com/sgiraz/homelog/internal/middleware"
    "github.com/sgiraz/homelog/internal/models"
)

type ExpenseHandler struct {
    db *gorm.DB
}

func NewExpenseHandler(db *gorm.DB) *ExpenseHandler {
    return &ExpenseHandler{db: db}
}

func (h *ExpenseHandler) List(c *gin.Context) {
    userID, _ := middleware.GetUserID(c)
    
    var expenses []models.Expense
    query := h.db.Where("user_id = ?", userID).
        Preload("Category").
        Preload("Property")
    
    // Filtri opzionali
    if categoryID := c.Query("category_id"); categoryID != "" {
        query = query.Where("category_id = ?", categoryID)
    }
    if from := c.Query("from"); from != "" {
        query = query.Where("date >= ?", from)
    }
    if to := c.Query("to"); to != "" {
        query = query.Where("date <= ?", to)
    }
    
    if err := query.Find(&expenses).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch expenses"})
        return
    }
    
    c.JSON(http.StatusOK, expenses)
}

// ... altri metodi
```

### Frontend Vue 3 (da creare)

**Struttura completa da creare in `/frontend/src/`:**

```
src/
├── main.js              # Entry point Vue
├── App.vue              # Componente root
├── router/
│   └── index.js         # Vue Router config
├── stores/
│   ├── auth.js          # Pinia store auth
│   ├── expenses.js      # Pinia store expenses
│   ├── utilities.js     # Pinia store utilities
│   └── projects.js      # Pinia store projects
├── views/
│   ├── LoginView.vue    # Login/Register
│   ├── DashboardView.vue # Dashboard principale
│   ├── ExpensesView.vue # Lista spese
│   ├── UtilitiesView.vue # Lista utilities
│   ├── ProjectsView.vue # Lista progetti
│   └── SettingsView.vue # Impostazioni
├── components/
│   ├── common/
│   │   ├── Button.vue
│   │   ├── Card.vue
│   │   ├── Modal.vue
│   │   └── Input.vue
│   ├── charts/
│   │   ├── BarChart.vue
│   │   └── PieChart.vue
│   └── layout/
│       ├── Navbar.vue
│       ├── Sidebar.vue
│       └── MobileNav.vue
├── api/
│   └── client.js        # Axios instance + API calls
├── utils/
│   ├── format.js        # Date/currency formatters
│   └── validators.js    # Form validators
├── assets/
│   └── styles/
│       └── main.css     # Tailwind imports
└── composables/
    ├── useAuth.js       # Auth composable
    └── useDarkMode.js   # Dark mode composable
```

**File essenziali da creare:**

1. **main.js**
```javascript
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './assets/styles/main.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
```

2. **api/client.js**
```javascript
import axios from 'axios'

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1',
  headers: {
    'Content-Type': 'application/json'
  }
})

// Interceptor per JWT token
apiClient.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Interceptor per refresh token
apiClient.interceptors.response.use(
  response => response,
  async error => {
    if (error.response?.status === 401) {
      // Handle token refresh
      const refreshToken = localStorage.getItem('refreshToken')
      if (refreshToken) {
        try {
          const { data } = await axios.post('/api/v1/auth/refresh', { refresh_token: refreshToken })
          localStorage.setItem('token', data.token)
          error.config.headers.Authorization = `Bearer ${data.token}`
          return axios(error.config)
        } catch {
          // Logout user
          localStorage.clear()
          window.location = '/login'
        }
      }
    }
    return Promise.reject(error)
  }
)

export default apiClient

// API methods
export const authAPI = {
  register: (data) => apiClient.post('/auth/register', data),
  login: (data) => apiClient.post('/auth/login', data),
}

export const expensesAPI = {
  list: (params) => apiClient.get('/expenses', { params }),
  create: (data) => apiClient.post('/expenses', data),
  update: (id, data) => apiClient.put(`/expenses/${id}`, data),
  delete: (id) => apiClient.delete(`/expenses/${id}`),
}

// ... altri API methods
```

3. **stores/auth.js** (Pinia)
```javascript
import { defineStore } from 'pinia'
import { authAPI } from '@/api/client'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null,
    token: localStorage.getItem('token'),
    isAuthenticated: false
  }),
  
  actions: {
    async login(credentials) {
      const { data } = await authAPI.login(credentials)
      this.user = data.user
      this.token = data.token
      this.isAuthenticated = true
      localStorage.setItem('token', data.token)
      localStorage.setItem('refreshToken', data.refresh_token)
    },
    
    logout() {
      this.user = null
      this.token = null
      this.isAuthenticated = false
      localStorage.clear()
    }
  }
})
```

4. **App.vue** (basato sul prototipo React)
```vue
<template>
  <div :class="{ 'dark': isDark }">
    <div class="min-h-screen bg-gray-50 dark:bg-gray-900 transition-colors">
      <Navbar />
      <div class="flex">
        <Sidebar v-if="isAuthenticated" />
        <main class="flex-1 p-6">
          <router-view />
        </main>
      </div>
      <MobileNav v-if="isAuthenticated" />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useDarkMode } from '@/composables/useDarkMode'
import Navbar from '@/components/layout/Navbar.vue'
import Sidebar from '@/components/layout/Sidebar.vue'
import MobileNav from '@/components/layout/MobileNav.vue'

const authStore = useAuthStore()
const { isDark } = useDarkMode()
const isAuthenticated = computed(() => authStore.isAuthenticated)
</script>
```

---

## 🚀 GUIDA COMPLETAMENTO PROGETTO

### Step 1: Completare Backend Handlers

1. Crea i file mancanti in `/backend/internal/handlers/`
2. Segui il template fornito sopra per `expense.go`
3. Replica per: `property.go`, `category.go`, `utility.go`, `project.go`, `settings.go`
4. Ogni handler deve:
   - Estrarre `userID` da context (middleware JWT)
   - Filtrare per `userID` per sicurezza
   - Validare input con binding
   - Gestire errori con status codes appropriati

### Step 2: Creare Frontend Vue 3

1. Inizializza il progetto:
```bash
cd frontend
npm install
```

2. Crea la struttura cartelle descritta sopra

3. Converti i componenti React del prototipo in Vue:
   - Usa `<script setup>` (Composition API)
   - Sostituisci `useState` con `ref` o `reactive`
   - Usa `@click` invece di `onClick`
   - Usa `v-if` invece di `{condition && <Component />}`

4. Esempio conversione Dashboard:
```vue
<template>
  <div class="space-y-6">
    <!-- KPI Cards -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <Card class="backdrop-blur-xl bg-opacity-70">
        <div class="flex items-center justify-between mb-2">
          <span class="text-gray-600 dark:text-gray-400 text-sm">Spese Mese</span>
          <Euro :size="20" class="text-blue-500" />
        </div>
        <div class="text-3xl font-bold text-gray-900 dark:text-white">
          {{ formatCurrency(monthlyExpenses) }}
        </div>
        <div class="text-sm text-green-500">+12% vs mese scorso</div>
        
        <!-- Ultima spesa -->
        <div class="pt-3 border-t border-gray-200 dark:border-gray-700 mt-3">
          <div class="text-xs text-gray-600 dark:text-gray-400 mb-2">Ultima spesa:</div>
          <div class="flex items-center justify-between">
            <div>
              <div class="text-sm font-medium">{{ lastExpense.description }}</div>
              <div class="text-xs text-gray-600">{{ lastExpense.date }} • {{ lastExpense.category }}</div>
            </div>
            <div class="text-sm font-bold text-blue-500">{{ formatCurrency(lastExpense.amount) }}</div>
          </div>
          <button @click="goToExpenses" class="mt-2 text-xs text-blue-500 hover:text-blue-600 flex items-center gap-1">
            Vedi tutte <ChevronRight :size="14" />
          </button>
        </div>
      </Card>
      <!-- ... altri KPI -->
    </div>

    <!-- Filtri unificati -->
    <Card class="p-4 backdrop-blur-xl bg-opacity-70">
      <div class="flex flex-col lg:flex-row gap-4">
        <div class="flex items-center gap-2">
          <Filter :size="20" class="text-gray-600" />
          <span class="font-medium">Filtri Grafici</span>
        </div>

        <!-- Preset temporali -->
        <div class="flex gap-2">
          <button
            v-for="preset in timePresets"
            :key="preset.value"
            @click="setTimePreset(preset.value)"
            :class="[
              'px-3 py-1.5 rounded-lg text-sm font-medium transition-colors',
              timePreset === preset.value
                ? 'bg-blue-500 text-white'
                : 'bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700'
            ]"
          >
            {{ preset.label }}
          </button>
        </div>

        <!-- Range personalizzato -->
        <div class="flex items-center gap-2">
          <input
            v-model="dateFrom"
            type="date"
            class="px-3 py-1.5 border rounded-lg text-sm"
          />
          <span>→</span>
          <input
            v-model="dateTo"
            type="date"
            class="px-3 py-1.5 border rounded-lg text-sm"
          />
        </div>

        <!-- Filtro categoria -->
        <select v-model="selectedCategory" class="px-3 py-1.5 border rounded-lg text-sm">
          <option value="all">📁 Tutte le categorie</option>
          <option value="casa">🏠 Casa</option>
          <option value="alimentari">🍕 Alimentari</option>
        </select>
      </div>
    </Card>

    <!-- Grafici -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Grafico spese mensili (2 colonne) -->
      <Card class="lg:col-span-2 p-6 backdrop-blur-xl bg-opacity-70">
        <h3 class="text-lg font-semibold mb-4">📈 Spese Mensili</h3>
        <BarChart v-if="filteredBarData.length > 0" :data="filteredBarData" />
        <EmptyState v-else icon="📊" message="Nessun dato disponibile" />
      </Card>

      <!-- Grafico categorie (1 colonna) -->
      <Card class="p-6 backdrop-blur-xl bg-opacity-70">
        <h3 class="text-lg font-semibold mb-4">🏠 Categorie</h3>
        <PieChart v-if="filteredPieData.length > 0" :data="filteredPieData" />
        <EmptyState v-else icon="🥧" message="Nessun dato" />
      </Card>
    </div>

    <!-- Prossime scadenze -->
    <Card class="p-6 backdrop-blur-xl bg-opacity-70">
      <h3 class="text-lg font-semibold mb-4">💡 Prossime Scadenze</h3>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
        <div
          v-for="bill in upcomingBills"
          :key="bill.id"
          class="flex items-center justify-between p-4 rounded-xl border"
          :class="`bg-${bill.colorClass}-50 dark:bg-${bill.colorClass}-900/20 border-${bill.colorClass}-200`"
        >
          <div class="flex items-center gap-3">
            <component :is="bill.icon" :size="24" :class="`text-${bill.colorClass}-500`" />
            <div>
              <div class="font-medium">{{ bill.name }}</div>
              <div class="text-sm text-gray-600">{{ formatDate(bill.dueDate) }}</div>
            </div>
          </div>
          <div :class="`font-bold text-${bill.colorClass}-600`">
            {{ formatCurrency(bill.amount) }}
          </div>
        </div>
      </div>
    </Card>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useExpensesStore } from '@/stores/expenses'
import { Euro, Filter, ChevronRight, Zap, Flame, Trash2 } from 'lucide-vue-next'
import Card from '@/components/common/Card.vue'
import BarChart from '@/components/charts/BarChart.vue'
import PieChart from '@/components/charts/PieChart.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { formatCurrency, formatDate } from '@/utils/format'

const router = useRouter()
const expensesStore = useExpensesStore()

// State
const timePreset = ref('6m')
const dateFrom = ref('2024-07-01')
const dateTo = ref('2024-12-31')
const selectedCategory = ref('all')

const timePresets = [
  { value: '1d', label: 'Oggi' },
  { value: '1m', label: '1 Mese' },
  { value: '3m', label: '3 Mesi' },
  { value: '6m', label: '6 Mesi' },
  { value: '1y', label: '1 Anno' }
]

// Computed
const monthlyExpenses = computed(() => expensesStore.monthlyTotal)
const lastExpense = computed(() => expensesStore.lastExpense)
const filteredBarData = computed(() => expensesStore.getFilteredBarData(selectedCategory.value))
const filteredPieData = computed(() => expensesStore.getFilteredPieData(selectedCategory.value))

const upcomingBills = [
  { id: 1, name: 'Bolletta Luce', dueDate: '2025-04-07', amount: 58.00, icon: Zap, colorClass: 'yellow' },
  { id: 2, name: 'Bolletta Gas', dueDate: '2025-04-13', amount: 160.00, icon: Flame, colorClass: 'orange' },
  { id: 3, name: 'Rifiuti ETRA', dueDate: '2025-05-16', amount: 351.32, icon: Trash2, colorClass: 'green' }
]

// Methods
const setTimePreset = (value) => {
  timePreset.value = value
  // Aggiorna dateFrom e dateTo in base al preset
}

const goToExpenses = () => {
  router.push('/expenses')
}
</script>
```

### Step 3: Testing

1. Testa backend:
```bash
cd backend
go test ./...
```

2. Testa frontend:
```bash
cd frontend
npm run dev
```

3. Testa end-to-end:
   - Registra un nuovo utente
   - Crea una proprietà
   - Aggiungi spese
   - Verifica grafici

### Step 4: Deploy su Raspberry Pi

1. **Preparazione Raspberry Pi:**
```bash
# Installa Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Aggiungi utente a gruppo docker
sudo usermod -aG docker $USER

# Riavvia per applicare modifiche
sudo reboot
```

2. **Deploy:**
```bash
# Clona repository sul Raspberry Pi
git clone https://github.com/sgiraz/homelog.git
cd homelog

# Configura .env
cp .env.example .env
nano .env  # Modifica JWT_SECRET e altre variabili

# Build e start
docker-compose up -d

# Verifica status
docker-compose ps
docker-compose logs -f
```

3. **Accesso:**
```
http://192.168.x.x:3000  # Frontend
http://192.168.x.x:8080  # Backend API
```

4. **Setup Tailscale (accesso remoto):**
```bash
# Installa Tailscale
curl -fsSL https://tailscale.com/install.sh | sh

# Connetti
sudo tailscale up

# Accedi da remoto
https://your-device.tailnet.ts.net:3000
```

---

## 📚 RISORSE UTILI

### Documentazione
- Go Gin: https://gin-gonic.com/docs/
- Vue 3: https://vuejs.org/guide/
- Pinia: https://pinia.vuejs.org/
- GORM: https://gorm.io/docs/
- Tailwind CSS: https://tailwindcss.com/docs

### Tools
- SQLite Browser: https://sqlitebrowser.org/
- Postman: https://www.postman.com/ (test API)
- Vue DevTools: https://devtools.vuejs.org/

---

## 🎯 PROSSIMI STEP SUGGERITI

### Fase 1 (Immediate)
1. ✅ Completare tutti gli handler backend mancanti
2. ✅ Creare struttura frontend Vue completa
3. ✅ Implementare login/register UI
4. ✅ Dashboard con grafici
5. ✅ Lista spese con filtri

### Fase 2 (Utilities)
1. ⏭️ Gestione utilities completa
2. ⏭️ Autoletture con upload foto
3. ⏭️ Bollette con PDF attachment
4. ⏭️ Alert automatici

### Fase 3 (Advanced)
1. ⏭️ Progetti di spesa
2. ⏭️ Budget mensili
3. ⏭️ Import/Export CSV
4. ⏭️ Report PDF

### Fase 4 (OCR & Community)
1. ⏭️ OCR bollette
2. ⏭️ Template parser community
3. ⏭️ Multi-lingua (i18n)
4. ⏭️ Plugin system

---

## 🐛 TROUBLESHOOTING COMUNE

### Backend non si avvia
```bash
# Verifica Go version
go version  # Deve essere >= 1.21

# Pulisci cache
go clean -modcache
go mod download

# Verifica database path
mkdir -p ./data
chmod 755 ./data
```

### Frontend build error
```bash
# Pulisci node_modules
rm -rf node_modules package-lock.json
npm install

# Verifica Node version
node --version  # Deve essere >= 20
```

### Docker build error su Raspberry Pi
```bash
# Verifica architettura
uname -m  # Deve essere armv7l o aarch64

# Build manualmente
cd backend
GOOS=linux GOARCH=arm GOARM=7 go build -o homelog-api ./cmd/api/main.go
```

### Database locked error
```bash
# Verifica che solo un'istanza acceda al DB
docker-compose down
rm ./data/homelog.db-wal ./data/homelog.db-shm
docker-compose up -d
```

---

## 📞 SUPPORTO

Se incontri problemi durante lo sviluppo, ricorda:

1. **Verifica i log:** `docker-compose logs -f`
2. **Controlla le issue:** GitHub Issues
3. **Leggi la documentazione:** docs/ folder
4. **Chiedi alla community:** GitHub Discussions

---

**Buon coding! 🚀**

*Documento creato: 27/01/2026*
*Versione: 1.0*
