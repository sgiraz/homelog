<template>
  <div class="space-y-4">
    <!-- Versione & Aggiornamenti -->
    <Card class="p-6">
      <h2 class="text-xl font-bold text-ink mb-4">{{ t('settings.data.versionTitle') }}</h2>
      <div class="flex items-center justify-between gap-4">
        <div>
          <div class="text-sm text-ink-soft">{{ t('settings.data.versionLabel') }}</div>
          <div class="text-lg font-semibold text-ink font-mono">{{ currentVersion }}</div>
        </div>
        <Button @click="checkForUpdates" :disabled="checking" variant="secondary" size="sm">
          <svg v-if="checking" class="w-4 h-4 mr-1.5 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
          </svg>
          {{ checking ? t('settings.data.checking') : t('settings.data.checkUpdates') }}
        </Button>
      </div>

      <!-- Update result -->
      <div v-if="updateResult" class="mt-4">
        <div v-if="updateResult.update_available" class="flex items-start gap-3 p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
          <svg class="w-5 h-5 text-green-600 dark:text-green-400 shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 11l5-5m0 0l5 5m-5-5v12" />
          </svg>
          <div>
            <div class="text-sm font-medium text-green-800 dark:text-green-200">
              {{ t('settings.data.updateAvailable', { version: updateResult.latest }) }}
            </div>
            <a
              :href="updateResult.latest_url"
              target="_blank"
              rel="noopener noreferrer"
              class="text-sm text-green-700 dark:text-green-300 underline hover:no-underline mt-1 inline-block"
            >
              {{ t('settings.data.viewChangelog') }}
            </a>
            <div class="text-xs text-green-600 dark:text-green-400 mt-2">
              {{ t('settings.data.updateInstructions') }} <code class="bg-green-100 dark:bg-green-800/50 px-1.5 py-0.5 rounded">docker compose pull &amp;&amp; docker compose up -d</code>
            </div>
          </div>
        </div>
        <div v-else class="flex items-center gap-2 p-3 bg-surface rounded-lg">
          <svg class="w-5 h-5 text-ink-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          <span class="text-sm text-ink-soft">{{ t('settings.data.upToDate') }}</span>
        </div>
      </div>

      <div v-if="checkError" class="mt-4 p-3 bg-red-50 dark:bg-red-900/20 rounded-lg text-sm text-red-700 dark:text-red-300">
        {{ t('settings.data.checkError') }}
      </div>
    </Card>

    <Card class="p-6">
      <h2 class="text-xl font-bold text-ink mb-4">{{ t('settings.data.backupTitle') }}</h2>

      <div class="space-y-6">
        <!-- Export -->
        <div>
          <h3 class="font-medium text-ink mb-1">{{ t('settings.data.exportTitle') }}</h3>
          <p class="text-sm text-ink-soft mb-3">
            {{ t('settings.data.exportDescription') }}
          </p>

          <!-- Format toggle -->
          <div class="flex items-center gap-1 mb-4 p-1 bg-surface-2 rounded-lg w-fit">
            <button
              @click="exportFormat = 'json'"
              :class="[
                'flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium whitespace-nowrap transition-colors',
                exportFormat === 'json'
                  ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
                  : 'text-ink-soft hover:bg-surface-2'
              ]"
            >
              JSON
            </button>
            <button
              @click="exportFormat = 'csv'"
              :class="[
                'flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium whitespace-nowrap transition-colors',
                exportFormat === 'csv'
                  ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
                  : 'text-ink-soft hover:bg-surface-2'
              ]"
            >
              CSV
            </button>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Button @click="doExport('all')" :disabled="exportLoading" class="w-full">
              {{ exportLoading === 'all' ? t('settings.data.exporting') : t('settings.data.exportAll') }}
            </Button>
            <Button @click="doExport('expenses')" :disabled="exportLoading" variant="secondary" class="w-full">
              {{ exportLoading === 'expenses' ? t('settings.data.exporting') : t('settings.data.exportExpenses') }}
            </Button>
            <Button @click="doExport('utilities')" :disabled="exportLoading" variant="secondary" class="w-full">
              {{ exportLoading === 'utilities' ? t('settings.data.exporting') : t('settings.data.exportUtilities') }}
            </Button>
            <Button @click="doExport('projects')" :disabled="exportLoading" variant="secondary" class="w-full">
              {{ exportLoading === 'projects' ? t('settings.data.exporting') : t('settings.data.exportProjects') }}
            </Button>
          </div>
        </div>

        <!-- Import -->
        <div class="border-t border-line pt-6">
          <h3 class="font-medium text-ink mb-1">{{ t('settings.data.importTitle') }}</h3>
          <p class="text-sm text-ink-soft mb-4">
            {{ t('settings.data.importDescription') }}
          </p>

          <div
            @dragover.prevent
            @dragenter.prevent="isDragging = true"
            @dragleave.prevent="isDragging = false"
            @drop.prevent="handleFileDrop"
            :class="[
              'border-2 border-dashed rounded-xl p-8 text-center transition-colors cursor-pointer',
              isDragging
                ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                : 'border-line hover:border-blue-400 dark:hover:border-blue-500'
            ]"
            @click="$refs.fileInput.click()"
          >
            <input ref="fileInput" type="file" accept=".json" class="hidden" @change="handleFileSelect" />
            <div class="text-3xl mb-2">📁</div>
            <p class="text-sm text-ink-soft">
              {{ t('settings.data.importDropPrompt') }} <span class="text-blue-600 dark:text-blue-400 underline">{{ t('settings.data.importDropSelect') }}</span>
            </p>
            <p v-if="selectedFile" class="text-sm font-medium text-ink mt-2">
              {{ selectedFile.name }}
            </p>
          </div>

          <div v-if="selectedFile" class="mt-4 space-y-3">
            <div class="flex items-start gap-2 p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg text-sm text-yellow-800 dark:text-yellow-200">
              <span class="shrink-0 font-bold">!</span>
              <span>{{ t('settings.data.importWarning') }}</span>
            </div>
            <Button @click="doImport" :disabled="importLoading" class="w-full">
              {{ importLoading ? t('settings.data.importing') : t('settings.data.importButton') }}
            </Button>
          </div>
        </div>
      </div>
    </Card>
  </div>
</template>

<script setup>
defineOptions({ name: 'DataTab' })

import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { exportAPI, versionAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'

const { t } = useI18n()
const currentVersion = __APP_VERSION__
const checking = ref(false)
const updateResult = ref(null)
const checkError = ref(false)

async function checkForUpdates() {
  checking.value = true
  checkError.value = false
  updateResult.value = null
  try {
    const { data } = await versionAPI.check()
    updateResult.value = data
  } catch {
    checkError.value = true
  } finally {
    checking.value = false
  }
}

const exportLoading = ref(null)
const exportFormat = ref('json')
const importLoading = ref(false)
const isDragging = ref(false)
const selectedFile = ref(null)
const fileInput = ref(null)

async function doExport(type) {
  exportLoading.value = type
  try {
    const isCSV = exportFormat.value === 'csv'
    const apiMap = {
      all: isCSV ? exportAPI.exportAllCSV : exportAPI.exportAll,
      expenses: isCSV ? exportAPI.exportExpensesCSV : exportAPI.exportExpenses,
      utilities: isCSV ? exportAPI.exportUtilitiesCSV : exportAPI.exportUtilities,
      projects: isCSV ? exportAPI.exportProjectsCSV : exportAPI.exportProjects,
    }
    const nameMap = {
      all: isCSV ? 'spese' : 'backup_completo',
      expenses: 'spese',
      utilities: 'utenze',
      projects: 'progetti',
    }
    const ext = isCSV ? 'csv' : 'json'
    const res = await apiMap[type]()
    const timestamp = new Date().toISOString().slice(0, 10)
    triggerDownload(res.data, `homelog_${nameMap[type]}_${timestamp}.${ext}`)
    window.$toast?.success(t('settings.data.exportSuccess'))
  } catch (err) {
    window.$toast?.error(t('settings.data.exportError', { error: err.response?.data?.error || err.message }))
  } finally {
    exportLoading.value = null
  }
}

function triggerDownload(blob, filename) {
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  window.URL.revokeObjectURL(url)
}

function handleFileDrop(e) {
  isDragging.value = false
  const file = e.dataTransfer.files[0]
  if (file && file.name.endsWith('.json')) {
    selectedFile.value = file
  } else {
    window.$toast?.error(t('settings.data.invalidFile'))
  }
}

function handleFileSelect(e) {
  const file = e.target.files[0]
  if (file) {
    selectedFile.value = file
  }
}

async function doImport() {
  if (!selectedFile.value) return
  importLoading.value = true
  try {
    const text = await selectedFile.value.text()
    const data = JSON.parse(text)
    const res = await exportAPI.importData(data)
    const counts = res.data.imported || {}
    const summary = Object.entries(counts)
      .map(([k, v]) => `${v} ${k}`)
      .join(', ')
    window.$toast?.success(t('settings.data.importSuccess', { summary: summary || t('settings.data.importEmpty') }))
    selectedFile.value = null
    if (fileInput.value) fileInput.value.value = ''
    setTimeout(() => { window.location.reload() }, 2000)
  } catch (err) {
    if (err instanceof SyntaxError) {
      window.$toast?.error(t('settings.data.invalidJson'))
    } else {
      window.$toast?.error(err.response?.data?.error || t('settings.data.importError', { error: err.message }))
    }
  } finally {
    importLoading.value = false
  }
}
</script>
