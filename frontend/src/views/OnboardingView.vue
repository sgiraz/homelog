<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 p-4">
    <div class="w-full max-w-2xl">
      <!-- Header -->
      <div class="text-center mb-6">
        <h1 class="text-3xl font-bold text-gray-900 dark:text-white">HomeLog</h1>
        <p class="text-gray-500 dark:text-gray-400 mt-1">Passo {{ currentStep }} di {{ totalSteps }}</p>
      </div>

      <!-- Step indicator -->
      <div class="flex items-center justify-center mb-8">
        <template v-for="(step, index) in steps" :key="step.id">
          <!-- Step circle -->
          <div class="flex flex-col items-center">
            <div
              :class="[
                'w-9 h-9 rounded-full flex items-center justify-center text-sm font-semibold transition-colors',
                currentStep > step.id
                  ? 'bg-green-500 text-white'
                  : currentStep === step.id
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400'
              ]"
            >
              <svg v-if="currentStep > step.id" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
              <span v-else>{{ step.id }}</span>
            </div>
            <span class="text-xs mt-1 text-gray-500 dark:text-gray-400 hidden sm:block">{{ step.label }}</span>
          </div>
          <!-- Connector line -->
          <div
            v-if="index < steps.length - 1"
            :class="[
              'flex-1 h-0.5 mx-2 transition-colors',
              currentStep > step.id ? 'bg-green-400' : 'bg-gray-200 dark:bg-gray-700'
            ]"
          />
        </template>
      </div>

      <!-- Card -->
      <Card className="p-6 sm:p-8">

        <!-- Step 1: Scelta percorso -->
        <div v-if="currentStep === 1">
          <div class="text-center mb-6">
            <div class="text-5xl mb-3">🏠</div>
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Come vuoi iniziare?</h2>
            <p class="text-gray-600 dark:text-gray-400 mt-2 text-sm leading-relaxed">
              Puoi creare la tua proprietà o unirti a una esistente
            </p>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-6">
            <!-- Crea -->
            <button
              @click="selectPath('create')"
              :class="[
                'flex flex-col items-center gap-3 p-6 rounded-xl border-2 transition-colors text-left',
                path === 'create'
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30'
                  : 'border-gray-200 dark:border-gray-700 hover:border-blue-300 dark:hover:border-blue-600 hover:bg-gray-50 dark:hover:bg-gray-700/50'
              ]"
            >
              <div class="flex items-center justify-center w-14 h-14 rounded-full bg-blue-100 dark:bg-blue-900/50">
                <svg class="w-7 h-7 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
                </svg>
              </div>
              <div>
                <p class="font-semibold text-gray-900 dark:text-white text-center">Crea una nuova casa</p>
                <p class="text-xs text-gray-500 dark:text-gray-400 mt-1 text-center">Configura la tua proprietà e invita i familiari</p>
              </div>
            </button>

            <!-- Unisciti -->
            <button
              @click="selectPath('join')"
              :class="[
                'flex flex-col items-center gap-3 p-6 rounded-xl border-2 transition-colors text-left',
                path === 'join'
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30'
                  : 'border-gray-200 dark:border-gray-700 hover:border-blue-300 dark:hover:border-blue-600 hover:bg-gray-50 dark:hover:bg-gray-700/50'
              ]"
            >
              <div class="flex items-center justify-center w-14 h-14 rounded-full bg-green-100 dark:bg-green-900/50">
                <svg class="w-7 h-7 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
              </div>
              <div>
                <p class="font-semibold text-gray-900 dark:text-white text-center">Unisciti a una casa esistente</p>
                <p class="text-xs text-gray-500 dark:text-gray-400 mt-1 text-center">Invia una richiesta a un amministratore</p>
              </div>
            </button>
          </div>

          <div class="flex justify-end mt-6">
            <Button class="w-full sm:w-auto px-8" @click="handleStep1" :disabled="!path">
              Avanti
            </Button>
          </div>
        </div>

        <!-- Step 2A: Crea proprietà -->
        <div v-else-if="currentStep === 2 && path === 'create'">
          <div class="text-center mb-6">
            <div class="text-5xl mb-3">🏠</div>
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Crea la tua proprietà</h2>
            <p class="text-gray-600 dark:text-gray-400 mt-2 text-sm">
              Inserisci i dettagli della tua casa.
            </p>
          </div>

          <div class="space-y-4">
            <Input
              v-model="property.name"
              label="Nome proprietà"
              placeholder="Es. Casa Milano, Appartamento Roma..."
              id="property-name"
            />
            <Input
              v-model="property.address"
              label="Indirizzo (opzionale)"
              placeholder="Es. Via Roma 1, Milano"
              id="property-address"
            />
          </div>

          <div v-if="stepError" class="mt-4 text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
            {{ stepError }}
          </div>

          <div class="flex gap-3 mt-6">
            <Button variant="secondary" class="flex-1" @click="currentStep = 1" :disabled="saving">
              Indietro
            </Button>
            <Button class="flex-1" @click="handleStep2Create" :disabled="saving || !property.name">
              {{ saving ? 'Salvataggio...' : 'Avanti' }}
            </Button>
          </div>
        </div>

        <!-- Step 2B: Richiesta join -->
        <div v-else-if="currentStep === 2 && path === 'join'">
          <div class="text-center mb-6">
            <div class="text-5xl mb-3">👥</div>
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Unisciti a una casa</h2>
            <p class="text-gray-600 dark:text-gray-400 mt-2 text-sm">
              Seleziona la proprietà a cui vuoi accedere.
            </p>
          </div>

          <!-- Loading -->
          <div v-if="loadingJoinable" class="py-8 text-center">
            <div class="inline-block w-6 h-6 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
            <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">Caricamento...</p>
          </div>

          <!-- No properties available -->
          <div v-else-if="joinableProperties.length === 0" class="py-6 text-center space-y-4">
            <div class="text-4xl">🔍</div>
            <p class="text-gray-600 dark:text-gray-400 text-sm">
              Nessuna casa trovata. Puoi crearne una tu!
            </p>
            <Button @click="switchToCreate">Crea la tua!</Button>
          </div>

          <!-- Properties list -->
          <div v-else class="space-y-3">
            <button
              v-for="prop in joinableProperties"
              :key="prop.id"
              @click="selectedPropertyId = prop.id"
              :class="[
                'w-full text-left p-4 rounded-xl border-2 transition-colors',
                selectedPropertyId === prop.id
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30'
                  : 'border-gray-200 dark:border-gray-700 hover:border-blue-300 dark:hover:border-blue-600 hover:bg-gray-50 dark:hover:bg-gray-700/50'
              ]"
            >
              <div class="flex items-center justify-between">
                <div>
                  <p class="font-semibold text-gray-900 dark:text-white">{{ prop.name }}</p>
                  <p v-if="prop.address" class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">{{ prop.address }}</p>
                  <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
                    {{ prop.residents ?? 1 }} {{ (prop.residents ?? 1) === 1 ? 'membro' : 'membri' }}
                  </p>
                </div>
                <div
                  :class="[
                    'w-5 h-5 rounded-full border-2 transition-colors flex-shrink-0',
                    selectedPropertyId === prop.id
                      ? 'border-blue-500 bg-blue-500'
                      : 'border-gray-300 dark:border-gray-600'
                  ]"
                >
                  <svg v-if="selectedPropertyId === prop.id" class="w-full h-full text-white p-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
                  </svg>
                </div>
              </div>
            </button>
          </div>

          <!-- Join request sent success -->
          <div v-if="joinRequestSent" class="mt-4 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
            <p class="text-green-700 dark:text-green-400 text-sm font-medium">
              Richiesta inviata! Un amministratore dovrà approvare la tua richiesta.
            </p>
          </div>

          <div v-if="stepError" class="mt-4 text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
            {{ stepError }}
          </div>

          <div v-if="joinableProperties.length > 0" class="flex gap-3 mt-6">
            <Button variant="secondary" class="flex-1" @click="currentStep = 1" :disabled="saving">
              Indietro
            </Button>
            <Button
              v-if="!joinRequestSent"
              class="flex-1"
              @click="handleStep2Join"
              :disabled="saving || !selectedPropertyId"
            >
              {{ saving ? 'Invio...' : 'Invia Richiesta' }}
            </Button>
            <Button v-else class="flex-1" @click="currentStep++">
              Avanti
            </Button>
          </div>
          <div v-else-if="!loadingJoinable" class="mt-4">
            <!-- empty: covered by switchToCreate button above -->
          </div>
        </div>

        <!-- Step 3: Preferenze -->
        <div v-else-if="currentStep === 3">
          <div class="text-center mb-6">
            <div class="text-5xl mb-3">⚙️</div>
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Preferenze</h2>
            <p class="text-gray-600 dark:text-gray-400 mt-2 text-sm">
              Configura lingua, valuta e tema dell'applicazione.
            </p>
          </div>

          <div class="space-y-5">
            <!-- Currency -->
            <div>
              <label class="block text-sm text-gray-600 dark:text-gray-400 mb-2">Valuta</label>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
                <button
                  v-for="c in currencies"
                  :key="c.value"
                  @click="preferences.currency = c.value"
                  :class="[
                    'px-3 py-2 rounded-lg text-sm font-medium border transition-colors',
                    preferences.currency === c.value
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                      : 'border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700'
                  ]"
                >
                  {{ c.symbol }} {{ c.value }}
                </button>
              </div>
            </div>

            <!-- Language -->
            <div>
              <label class="block text-sm text-gray-600 dark:text-gray-400 mb-2">Lingua</label>
              <div class="grid grid-cols-2 gap-2">
                <button
                  v-for="l in languages"
                  :key="l.value"
                  @click="preferences.language = l.value"
                  :class="[
                    'px-3 py-2 rounded-lg text-sm font-medium border transition-colors',
                    preferences.language === l.value
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                      : 'border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700'
                  ]"
                >
                  {{ l.label }}
                </button>
              </div>
            </div>

            <!-- Theme -->
            <div>
              <label class="block text-sm text-gray-600 dark:text-gray-400 mb-2">Tema</label>
              <div class="grid grid-cols-3 gap-2">
                <button
                  v-for="t in themes"
                  :key="t.value"
                  @click="preferences.theme = t.value"
                  :class="[
                    'px-3 py-2 rounded-lg text-sm font-medium border transition-colors',
                    preferences.theme === t.value
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                      : 'border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700'
                  ]"
                >
                  {{ t.icon }} {{ t.label }}
                </button>
              </div>
            </div>
          </div>

          <div v-if="stepError" class="mt-4 text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
            {{ stepError }}
          </div>

          <div class="flex gap-3 mt-6">
            <Button variant="secondary" class="flex-1" @click="skipStep" :disabled="saving">
              Salta
            </Button>
            <Button class="flex-1" @click="handleStep3" :disabled="saving">
              {{ saving ? 'Salvataggio...' : 'Avanti' }}
            </Button>
          </div>
        </div>

        <!-- Step 4: Scopri Funzionalità -->
        <div v-else-if="currentStep === 4">
          <div class="text-center mb-6">
            <div class="text-5xl mb-3">✨</div>
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Scopri HomeLog</h2>
            <p class="text-gray-600 dark:text-gray-400 mt-2 text-sm">
              Ecco cosa puoi fare con la tua nuova app.
            </p>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-6">
            <div
              v-for="feature in features"
              :key="feature.title"
              class="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-xl border border-gray-100 dark:border-gray-700"
            >
              <div class="text-2xl mb-2">{{ feature.icon }}</div>
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-1">{{ feature.title }}</h3>
              <p class="text-xs text-gray-500 dark:text-gray-400 leading-relaxed">{{ feature.description }}</p>
            </div>
          </div>

          <div v-if="stepError" class="mb-4 text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
            {{ stepError }}
          </div>

          <Button class="w-full" @click="completeOnboarding" :disabled="saving">
            {{ saving ? 'Completamento...' : 'Inizia!' }}
          </Button>
        </div>
      </Card>

      <!-- Back button (not on step 1) -->
      <div v-if="currentStep > 1" class="text-center mt-4">
        <button
          @click="currentStep--"
          class="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
        >
          ← Torna indietro
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { propertiesAPI, settingsAPI, joinRequestAPI } from '@/api/client'
import { useSettingsStore } from '@/stores/settings'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import Input from '@/components/common/Input.vue'

defineOptions({ name: 'OnboardingView' })

const router = useRouter()
const settingsStore = useSettingsStore()

const totalSteps = 4
const currentStep = ref(1)
const saving = ref(false)
const stepError = ref(null)

// Step 1 — path selection
const path = ref(null) // 'create' | 'join'

// Step 2A — create property
const property = ref({ name: '', address: '' })
const createdPropertyId = ref(null)

// Step 2B — join request
const joinableProperties = ref([])
const loadingJoinable = ref(false)
const selectedPropertyId = ref(null)
const joinRequestSent = ref(false)

// Step 3 — preferences
const preferences = ref({
  currency: 'EUR',
  language: 'it',
  theme: 'auto'
})

// Step labels depend on path
const steps = computed(() => [
  { id: 1, label: 'Percorso' },
  { id: 2, label: path.value === 'join' ? 'Unisciti' : 'Proprietà' },
  { id: 3, label: 'Preferenze' },
  { id: 4, label: 'Scopri' }
])

const currencies = [
  { value: 'EUR', symbol: '€' },
  { value: 'USD', symbol: '$' },
  { value: 'GBP', symbol: '£' },
  { value: 'CHF', symbol: 'Fr' }
]

const languages = [
  { value: 'it', label: '🇮🇹 Italiano' },
  { value: 'en', label: '🇬🇧 English' }
]

const themes = [
  { value: 'auto', label: 'Auto', icon: '💻' },
  { value: 'light', label: 'Chiaro', icon: '☀️' },
  { value: 'dark', label: 'Scuro', icon: '🌙' }
]

const features = [
  {
    icon: '📄',
    title: 'Estrazione PDF bollette',
    description: 'Carica le tue bollette in PDF e HomeLog estrae automaticamente i dati grazie ai template configurabili.'
  },
  {
    icon: '⚖️',
    title: 'Divisione spese',
    description: 'Suddividi le spese tra i membri della famiglia in modo equo e tieni traccia dei saldi.'
  },
  {
    icon: '📊',
    title: 'Analisi consumi utenze',
    description: 'Confronta le letture del contatore con quelle del fornitore e monitora i consumi nel tempo.'
  },
  {
    icon: '🏗️',
    title: 'Gestione progetti',
    description: 'Pianifica ristrutturazioni, acquisti o viaggi con budget dedicati e tracciamento delle spese.'
  }
]

onMounted(async () => {
  if (!settingsStore.loaded) {
    try {
      await settingsStore.loadSettings()
    } catch {
      // ignore
    }
  }
  if (settingsStore.onboardingCompleted) {
    router.replace('/')
    return
  }
  // Pre-fill preferences from current settings
  preferences.value.currency = settingsStore.currency || 'EUR'
  preferences.value.language = settingsStore.language || 'it'
  preferences.value.theme = settingsStore.theme || 'auto'
})

// Fetch joinable properties when entering join step
watch(currentStep, async (step) => {
  if (step === 2 && path.value === 'join') {
    await fetchJoinableProperties()
  }
})

async function fetchJoinableProperties() {
  loadingJoinable.value = true
  stepError.value = null
  try {
    const { data } = await joinRequestAPI.joinableProperties()
    joinableProperties.value = data || []
  } catch (err) {
    console.error('Error fetching joinable properties:', err)
    joinableProperties.value = []
  } finally {
    loadingJoinable.value = false
  }
}

function selectPath(chosen) {
  path.value = chosen
}

function switchToCreate() {
  path.value = 'create'
  currentStep.value = 2
}

function skipStep() {
  stepError.value = null
  currentStep.value++
}

// Step 1: path selection — no skip allowed
function handleStep1() {
  if (!path.value) return
  stepError.value = null
  currentStep.value = 2
}

// Step 2A: create property
async function handleStep2Create() {
  stepError.value = null
  if (!property.value.name.trim()) {
    currentStep.value++
    return
  }

  saving.value = true
  try {
    const { data } = await propertiesAPI.create({
      name: property.value.name.trim(),
      address: property.value.address.trim() || undefined,
      type: 'owned',
      is_current: true,
      residents: 1,
      start_date: new Date().toISOString()
    })
    createdPropertyId.value = data.id
    currentStep.value++
  } catch (err) {
    stepError.value = err.response?.data?.error || 'Errore nella creazione della proprietà.'
    window.$toast?.error(stepError.value)
  } finally {
    saving.value = false
  }
}

// Step 2B: send join request
async function handleStep2Join() {
  if (!selectedPropertyId.value) return
  stepError.value = null
  saving.value = true
  try {
    await joinRequestAPI.create(selectedPropertyId.value)
    joinRequestSent.value = true
    window.$toast?.success('Richiesta inviata con successo!')
  } catch (err) {
    stepError.value = err.response?.data?.error || 'Errore nell\'invio della richiesta.'
    window.$toast?.error(stepError.value)
  } finally {
    saving.value = false
  }
}

// Step 3: save preferences
async function handleStep3() {
  stepError.value = null
  saving.value = true
  try {
    await settingsStore.updateSettings({
      currency: preferences.value.currency,
      language: preferences.value.language,
      theme: preferences.value.theme
    })
    currentStep.value++
  } catch (err) {
    stepError.value = err.response?.data?.error || 'Errore nel salvataggio delle preferenze.'
    window.$toast?.error(stepError.value)
  } finally {
    saving.value = false
  }
}

// Step 4: complete onboarding
async function completeOnboarding() {
  stepError.value = null
  saving.value = true
  try {
    await settingsAPI.update({ onboarding_completed: true })
    settingsStore.onboardingCompleted = true
    router.push('/')
  } catch (err) {
    stepError.value = err.response?.data?.error || 'Errore nel completamento dell\'onboarding.'
    window.$toast?.error(stepError.value)
  } finally {
    saving.value = false
  }
}
</script>
