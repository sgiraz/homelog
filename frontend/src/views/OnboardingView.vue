<template>
  <div class="min-h-screen flex items-center justify-center bg-canvas p-4">
    <div class="w-full max-w-2xl">
      <!-- Header -->
      <div class="text-center mb-6">
        <h1 class="text-3xl font-bold text-ink">HomeLog</h1>
        <p class="text-ink-muted mt-1">{{ t('onboarding.stepIndicator', { current: currentStep, total: totalSteps }) }}</p>
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
                    : 'bg-surface-3 text-ink-muted'
              ]"
            >
              <svg v-if="currentStep > step.id" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
              </svg>
              <span v-else>{{ step.id }}</span>
            </div>
            <span class="text-xs mt-1 text-ink-muted hidden sm:block">{{ step.label }}</span>
          </div>
          <!-- Connector line -->
          <div
            v-if="index < steps.length - 1"
            :class="[
              'flex-1 h-0.5 mx-2 transition-colors',
              currentStep > step.id ? 'bg-green-400' : 'bg-surface-3'
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
            <h2 class="text-2xl font-bold text-ink">{{ t('onboarding.step1.title') }}</h2>
            <p class="text-ink-soft mt-2 text-sm leading-relaxed">
              {{ t('onboarding.step1.subtitle') }}
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
                  : 'border-line hover:border-blue-300 dark:hover:border-blue-600 hover:bg-surface-2'
              ]"
            >
              <div class="flex items-center justify-center w-14 h-14 rounded-full bg-blue-100 dark:bg-blue-900/50">
                <svg class="w-7 h-7 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
                </svg>
              </div>
              <div>
                <p class="font-semibold text-ink text-center">{{ t('onboarding.step1.createTitle') }}</p>
                <p class="text-xs text-ink-muted mt-1 text-center">{{ t('onboarding.step1.createDescription') }}</p>
              </div>
            </button>

            <!-- Unisciti -->
            <button
              @click="selectPath('join')"
              :class="[
                'flex flex-col items-center gap-3 p-6 rounded-xl border-2 transition-colors text-left',
                path === 'join'
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30'
                  : 'border-line hover:border-blue-300 dark:hover:border-blue-600 hover:bg-surface-2'
              ]"
            >
              <div class="flex items-center justify-center w-14 h-14 rounded-full bg-green-100 dark:bg-green-900/50">
                <svg class="w-7 h-7 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
              </div>
              <div>
                <p class="font-semibold text-ink text-center">{{ t('onboarding.step1.joinTitle') }}</p>
                <p class="text-xs text-ink-muted mt-1 text-center">{{ t('onboarding.step1.joinDescription') }}</p>
              </div>
            </button>
          </div>

          <div class="flex justify-end mt-6">
            <Button class="w-full sm:w-auto px-8" @click="handleStep1" :disabled="!path">
              {{ t('onboarding.nextButton') }}
            </Button>
          </div>
        </div>

        <!-- Step 2A: Crea proprietà -->
        <div v-else-if="currentStep === 2 && path === 'create'">
          <div class="text-center mb-6">
            <div class="text-5xl mb-3">🏠</div>
            <h2 class="text-2xl font-bold text-ink">{{ t('onboarding.step2Create.title') }}</h2>
            <p class="text-ink-soft mt-2 text-sm">
              {{ t('onboarding.step2Create.subtitle') }}
            </p>
          </div>

          <div class="space-y-4">
            <Input
              v-model="property.name"
              :label="t('onboarding.step2Create.nameLabel')"
              :placeholder="t('onboarding.step2Create.namePlaceholder')"
              id="property-name"
            />
            <Input
              v-model="property.address"
              :label="t('onboarding.step2Create.addressLabel')"
              :placeholder="t('onboarding.step2Create.addressPlaceholder')"
              id="property-address"
            />
          </div>

          <div v-if="stepError" class="mt-4 text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
            {{ stepError }}
          </div>

          <div class="flex gap-3 mt-6">
            <Button variant="secondary" class="flex-1" @click="currentStep = 1" :disabled="saving">
              {{ t('onboarding.previousButton') }}
            </Button>
            <Button class="flex-1" @click="handleStep2Create" :disabled="saving || !property.name">
              {{ saving ? t('onboarding.savingButton') : t('onboarding.nextButton') }}
            </Button>
          </div>
        </div>

        <!-- Step 2B: Richiesta join -->
        <div v-else-if="currentStep === 2 && path === 'join'">
          <div class="text-center mb-6">
            <div class="text-5xl mb-3">👥</div>
            <h2 class="text-2xl font-bold text-ink">{{ t('onboarding.step2Join.title') }}</h2>
            <p class="text-ink-soft mt-2 text-sm">
              {{ t('onboarding.step2Join.subtitle') }}
            </p>
          </div>

          <!-- Loading -->
          <div v-if="loadingJoinable" class="py-8 text-center">
            <div class="inline-block w-6 h-6 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
            <p class="mt-2 text-sm text-ink-muted">{{ t('onboarding.step2Join.loading') }}</p>
          </div>

          <!-- No properties available -->
          <div v-else-if="joinableProperties.length === 0" class="py-6 text-center space-y-4">
            <div class="text-4xl">🔍</div>
            <p class="text-ink-soft text-sm">
              {{ t('onboarding.step2Join.noProperties') }}
            </p>
            <Button @click="switchToCreate">{{ t('onboarding.step2Join.createMine') }}</Button>
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
                  : 'border-line hover:border-blue-300 dark:hover:border-blue-600 hover:bg-surface-2'
              ]"
            >
              <div class="flex items-center justify-between">
                <div>
                  <p class="font-semibold text-ink">{{ prop.name }}</p>
                  <p v-if="prop.address" class="text-sm text-ink-muted mt-0.5">{{ prop.address }}</p>
                  <p class="text-xs text-ink-faint mt-1">
                    {{ t(`onboarding.step2Join.${(prop.residents ?? 1) === 1 ? 'memberCount_one' : 'memberCount_other'}`, { n: prop.residents ?? 1 }) }}
                  </p>
                </div>
                <div
                  :class="[
                    'w-5 h-5 rounded-full border-2 transition-colors flex-shrink-0',
                    selectedPropertyId === prop.id
                      ? 'border-blue-500 bg-blue-500'
                      : 'border-line'
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
              {{ t('onboarding.step2Join.requestSent') }}
            </p>
          </div>

          <div v-if="stepError" class="mt-4 text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
            {{ stepError }}
          </div>

          <div v-if="joinableProperties.length > 0" class="flex gap-3 mt-6">
            <Button variant="secondary" class="flex-1" @click="currentStep = 1" :disabled="saving">
              {{ t('onboarding.previousButton') }}
            </Button>
            <Button
              v-if="!joinRequestSent"
              class="flex-1"
              @click="handleStep2Join"
              :disabled="saving || !selectedPropertyId"
            >
              {{ saving ? t('onboarding.step2Join.submittingButton') : t('onboarding.step2Join.submitButton') }}
            </Button>
            <Button v-else class="flex-1" @click="currentStep++">
              {{ t('onboarding.nextButton') }}
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
            <h2 class="text-2xl font-bold text-ink">{{ t('onboarding.step3.title') }}</h2>
            <p class="text-ink-soft mt-2 text-sm">
              {{ t('onboarding.step3.subtitle') }}
            </p>
          </div>

          <div class="space-y-5">
            <!-- Currency -->
            <div>
              <label class="block text-sm text-ink-soft mb-2">{{ t('onboarding.step3.currencyLabel') }}</label>
              <select
                v-model="preferences.currency"
                class="w-full px-3 py-3 border border-line rounded-lg
                       bg-surface text-ink text-base
                       focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option v-for="c in currencies" :key="c.value" :value="c.value">
                  {{ c.symbol }} {{ c.value }} — {{ c.name }}
                </option>
              </select>
            </div>

            <!-- Language -->
            <div>
              <label class="block text-sm text-ink-soft mb-2">{{ t('onboarding.step3.languageLabel') }}</label>
              <div class="grid grid-cols-2 gap-2">
                <button
                  v-for="l in languages"
                  :key="l.value"
                  @click="preferences.language = l.value"
                  :class="[
                    'px-3 py-2 rounded-lg text-sm font-medium border transition-colors',
                    preferences.language === l.value
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                      : 'border-line text-ink-soft hover:bg-surface-2'
                  ]"
                >
                  {{ l.label }}
                </button>
              </div>
            </div>

            <!-- Theme -->
            <div>
              <label class="block text-sm text-ink-soft mb-2">{{ t('onboarding.step3.themeLabel') }}</label>
              <div class="grid grid-cols-3 gap-2">
                <button
                  v-for="th in themes"
                  :key="th.value"
                  @click="preferences.theme = th.value"
                  :class="[
                    'px-3 py-2 rounded-lg text-sm font-medium border transition-colors',
                    preferences.theme === th.value
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                      : 'border-line text-ink-soft hover:bg-surface-2'
                  ]"
                >
                  {{ th.icon }} {{ th.label }}
                </button>
              </div>
            </div>
          </div>

          <div v-if="stepError" class="mt-4 text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
            {{ stepError }}
          </div>

          <div class="flex gap-3 mt-6">
            <Button variant="secondary" class="flex-1" @click="skipStep" :disabled="saving">
              {{ t('onboarding.skipButton') }}
            </Button>
            <Button class="flex-1" @click="handleStep3" :disabled="saving">
              {{ saving ? t('onboarding.savingButton') : t('onboarding.nextButton') }}
            </Button>
          </div>
        </div>

        <!-- Step 4: Scopri Funzionalità -->
        <div v-else-if="currentStep === 4">
          <div class="text-center mb-6">
            <div class="text-5xl mb-3">✨</div>
            <h2 class="text-2xl font-bold text-ink">{{ t('onboarding.step4.title') }}</h2>
            <p class="text-ink-soft mt-2 text-sm">
              {{ t('onboarding.step4.subtitle') }}
            </p>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-6">
            <div
              v-for="feature in features"
              :key="feature.key"
              class="p-4 bg-surface-2/50 rounded-xl border border-line"
            >
              <div class="text-2xl mb-2">{{ feature.icon }}</div>
              <h3 class="text-sm font-semibold text-ink mb-1">{{ t(`onboarding.step4.features.${feature.key}.title`) }}</h3>
              <p class="text-xs text-ink-muted leading-relaxed">{{ t(`onboarding.step4.features.${feature.key}.description`) }}</p>
            </div>
          </div>

          <div v-if="stepError" class="mb-4 text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
            {{ stepError }}
          </div>

          <Button class="w-full" @click="completeOnboarding" :disabled="saving">
            {{ saving ? t('onboarding.step4.completingButton') : t('onboarding.step4.completeButton') }}
          </Button>
        </div>
      </Card>

      <!-- Back button (not on step 1) -->
      <div v-if="currentStep > 1" class="text-center mt-4">
        <button
          @click="currentStep--"
          class="text-sm text-ink-muted hover:text-ink-soft"
        >
          {{ t('onboarding.back') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { propertiesAPI, settingsAPI, joinRequestAPI } from '@/api/client'
import { currencies as allCurrencies } from '@/utils/currencies'
import { useSettingsStore } from '@/stores/settings'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import Input from '@/components/common/Input.vue'

defineOptions({ name: 'OnboardingView' })

const router = useRouter()
const { t } = useI18n()
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
  { id: 1, label: t('onboarding.steps.path') },
  { id: 2, label: path.value === 'join' ? t('onboarding.steps.join') : t('onboarding.steps.property') },
  { id: 3, label: t('onboarding.steps.preferences') },
  { id: 4, label: t('onboarding.steps.discover') }
])

const currencies = allCurrencies.map(c => ({ value: c.code, symbol: c.symbol, name: c.name }))

const languages = [
  { value: 'it', label: '🇮🇹 Italiano' },
  { value: 'en', label: '🇬🇧 English' }
]

const themes = computed(() => [
  { value: 'auto', label: t('onboarding.step3.themeAuto'), icon: '💻' },
  { value: 'light', label: t('onboarding.step3.themeLight'), icon: '☀️' },
  { value: 'dark', label: t('onboarding.step3.themeDark'), icon: '🌙' }
])

// Feature keys (icon stays in code, copy lives in i18n).
const features = [
  { key: 'pdf', icon: '📄' },
  { key: 'split', icon: '⚖️' },
  { key: 'consumption', icon: '📊' },
  { key: 'projects', icon: '🏗️' }
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
    stepError.value = t('onboarding.step2Create.nameRequired')
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
    stepError.value = err.response?.data?.error || t('onboarding.step2Create.createError')
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
    window.$toast?.success(t('onboarding.step2Join.submitSuccess'))
  } catch (err) {
    stepError.value = err.response?.data?.error || t('onboarding.step2Join.submitError')
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
    stepError.value = err.response?.data?.error || t('onboarding.step3.saveError')
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
    // Reload settings to pick up hasProperty and other server-side state
    await settingsStore.loadSettings()
    router.push('/')
  } catch (err) {
    stepError.value = err.response?.data?.error || t('onboarding.step4.completeError')
    window.$toast?.error(stepError.value)
  } finally {
    saving.value = false
  }
}
</script>
