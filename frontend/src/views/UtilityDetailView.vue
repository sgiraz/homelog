<template>
  <div class="space-y-4">
    <!-- Back + Header -->
    <div class="flex items-center gap-3">
      <button
        @click="goBack"
        class="p-2 -ml-2 rounded-lg hover:bg-surface-2 transition-colors"
        :aria-label="t('utilities.detail.back')"
      >
        <svg class="w-5 h-5 text-ink-soft" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>
      <div v-if="utility" class="flex items-center gap-3 flex-1 min-w-0">
        <div :class="['p-2.5 rounded-xl border flex-shrink-0', utilityColorClass]">
          <span class="text-xl">{{ utilityIcon }}</span>
        </div>
        <div class="min-w-0">
          <h1 class="text-xl sm:text-2xl font-bold text-ink truncate">{{ utility.provider }}</h1>
          <p class="text-sm text-ink-muted">{{ utilityTypeLabel }}</p>
        </div>
      </div>
      <!-- Actions -->
      <div v-if="utility" class="flex items-center gap-1 flex-shrink-0">
        <a
          v-if="utility.customer_portal"
          :href="utility.customer_portal"
          target="_blank"
          rel="noopener noreferrer"
          class="p-2.5 rounded-lg hover:bg-surface-2 text-ink-muted"
          :title="t('utilities.detail.openPortal')"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </a>
        <button
          v-if="settingsStore.isPropertyAdmin"
          @click="showEditModal = true"
          class="p-2.5 rounded-lg hover:bg-surface-2 text-ink-muted"
          :title="t('utilities.detail.editService')"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
          </svg>
        </button>
        <button
          v-if="settingsStore.isPropertyAdmin"
          @click="confirmDeleteUtility"
          class="p-2.5 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 text-ink-faint hover:text-red-500 dark:hover:text-red-400"
          :title="t('utilities.detail.deleteService')"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12 text-ink-muted">
      <div class="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
      {{ t('utilities.loading') }}
    </div>

    <template v-else-if="utility">
      <!-- Info Cards -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div v-if="utility.service_code" class="p-3 bg-surface rounded-xl border border-line">
          <div class="text-xs text-ink-muted mb-1">{{ serviceCodeLabel }}</div>
          <div class="text-sm font-medium text-ink truncate">{{ utility.service_code }}</div>
        </div>
        <div v-if="utility.customer_code" class="p-3 bg-surface rounded-xl border border-line">
          <div class="text-xs text-ink-muted mb-1">{{ isMetered ? t('utilities.detail.infoCustomer') : t('utilities.detail.infoContract') }}</div>
          <div class="text-sm font-medium text-ink truncate">{{ utility.customer_code }}</div>
        </div>
        <div v-if="utility.power_capacity" class="p-3 bg-surface rounded-xl border border-line">
          <div class="text-xs text-ink-muted mb-1">{{ t('utilities.detail.infoPower') }}</div>
          <div class="text-sm font-medium text-ink">{{ utility.power_capacity }} kW</div>
        </div>
        <div v-if="utility.recurring_amount" class="p-3 bg-surface rounded-xl border border-line">
          <div class="text-xs text-ink-muted mb-1">{{ t('utilities.detail.infoFee') }}</div>
          <div class="text-sm font-medium text-ink">{{ t('utilities.detail.infoFeePerUnit', { amount: formatCurrency(utility.recurring_amount), unit: billingFrequencyLabel }) }}</div>
        </div>
        <div v-if="utility.paid_by_member?.name" class="p-3 bg-surface rounded-xl border border-line">
          <div class="text-xs text-ink-muted mb-1">{{ t('utilities.detail.infoPaidBy') }}</div>
          <div class="text-sm font-medium text-ink truncate">{{ utility.paid_by_member.name }}</div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="overflow-x-auto -mx-4 sm:mx-0 px-4 sm:px-0 pb-1">
        <div class="flex gap-1 min-w-max sm:min-w-0">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            @click="activeTab = tab.id"
            :class="[
              'flex items-center gap-1.5 px-3 py-2.5 rounded-lg text-sm font-medium whitespace-nowrap transition-colors',
              activeTab === tab.id
                ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
                : 'text-ink-soft hover:bg-surface-2'
            ]"
          >
            <span>{{ tab.icon }}</span>
            <span>{{ tab.label }}</span>
            <span v-if="tab.count != null" class="text-xs opacity-70">({{ tab.count }})</span>
          </button>
        </div>
      </div>

      <!-- Tab Content -->
      <BillsTab
        v-show="activeTab === 'bills'"
        :utility="utility"
        :consumption-unit="consumptionUnit"
        @bill-saved="onDataChanged"
        @bill-deleted="onDataChanged"
        @bill-updated="onDataChanged"
      />

      <ReadingsTab
        v-show="activeTab === 'readings'"
        v-if="isMetered"
        :utility="utility"
        :consumption-unit="consumptionUnit"
        :active="activeTab === 'readings'"
        @reading-saved="onDataChanged"
        @reading-deleted="onDataChanged"
      />

      <AnalysisTab
        v-show="activeTab === 'analysis'"
        v-if="isMetered"
        ref="analysisTab"
        :utility="utility"
        :consumption-unit="consumptionUnit"
        @threshold-saved="onThresholdSaved"
      />

      <PriceHistoryTab
        v-show="activeTab === 'price_history'"
        v-if="!isMetered"
        :utility="utility"
      />
    </template>

    <!-- Edit utility modal -->
    <EditUtilityModal
      v-if="showEditModal"
      :utility="utility"
      @close="showEditModal = false"
      @updated="onUtilityUpdated"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUtilitiesStore } from '@/stores/utilities'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import BillsTab from '@/components/utilities/BillsTab.vue'
import ReadingsTab from '@/components/utilities/ReadingsTab.vue'
import AnalysisTab from '@/components/utilities/AnalysisTab.vue'
import PriceHistoryTab from '@/components/utilities/PriceHistoryTab.vue'
import EditUtilityModal from '@/components/utilities/EditUtilityModal.vue'

defineOptions({ name: 'UtilityDetailView' })

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const utilitiesStore = useUtilitiesStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

const loading = ref(true)
const utility = ref(null)
// Deep-link from global search lands here with `?tab=bills&highlight=<id>`;
// the tab query wins over the default.
const validTabs = ['bills', 'readings', 'analysis', 'price_history']
const activeTab = ref(validTabs.includes(route.query.tab) ? route.query.tab : 'bills')
const showEditModal = ref(false)

// When navigating to the same utility with a different ?tab query (e.g. from
// global search deep-link while the view is already mounted), update activeTab.
watch(() => route.query.tab, (tab) => {
  if (tab && validTabs.includes(tab)) {
    activeTab.value = tab
  }
})
const analysisTab = ref(null)

// ── Computed ──

const isMetered = computed(() => {
  return ['electricity', 'gas', 'water'].includes(utility.value?.type)
})

const billingFrequencyLabel = computed(() => {
  const n = utility.value?.billing_interval || 1
  const u = utility.value?.billing_unit || 'month'
  const suffix = n === 1 ? '_one' : '_other'
  const key = `utilities.detail.billingFrequency.${u}${suffix}`
  const word = t(key)
  const fallback = key === word ? u : word
  return n === 1 ? fallback : t('utilities.detail.billingFrequency.compact', { n, unit: fallback })
})

const tabs = computed(() => {
  const billLabel = isMetered.value ? t('utilities.detail.tabs.bills') : t('utilities.detail.tabs.invoices')
  const list = [
    { id: 'bills', label: billLabel, icon: '\uD83D\uDCC4', count: utility.value?.bills?.length || 0 },
  ]
  if (isMetered.value) {
    list.push({ id: 'readings', label: t('utilities.detail.tabs.readings'), icon: '\uD83D\uDCCA', count: utility.value?.readings?.length || 0 })
    list.push({ id: 'analysis', label: t('utilities.detail.tabs.analysis'), icon: '\uD83D\uDCC8', count: null })
  } else {
    list.push({ id: 'price_history', label: t('utilities.detail.tabs.priceHistory'), icon: '\uD83D\uDCC8', count: utility.value?.price_changes?.length || 0 })
  }
  return list
})

const utilityIcon = computed(() => {
  const icons = {
    electricity: '\u26A1', gas: '\uD83D\uDD25', water: '\uD83D\uDCA7', waste: '\u267B\uFE0F',
    internet: '\uD83C\uDF10', insurance: '\uD83D\uDEE1\uFE0F', affitto: '\uD83C\uDFE0', mutuo: '\uD83C\uDFE6'
  }
  return icons[utility.value?.type] || '\u26A1'
})

const utilityTypeLabel = computed(() => {
  if (!utility.value?.type) return ''
  const key = `utilities.utilityTypes.${utility.value.type}`
  const label = t(key)
  return label === key ? utility.value.type : label
})

const utilityColorClass = computed(() => {
  const classes = {
    electricity: 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800',
    gas: 'bg-orange-50 dark:bg-orange-900/20 border-orange-200 dark:border-orange-800',
    water: 'bg-cyan-50 dark:bg-cyan-900/20 border-cyan-200 dark:border-cyan-800',
    waste: 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800',
    internet: 'bg-indigo-50 dark:bg-indigo-900/20 border-indigo-200 dark:border-indigo-800',
    insurance: 'bg-emerald-50 dark:bg-emerald-900/20 border-emerald-200 dark:border-emerald-800',
    affitto: 'bg-purple-50 dark:bg-purple-900/20 border-purple-200 dark:border-purple-800',
    mutuo: 'bg-sky-50 dark:bg-sky-900/20 border-sky-200 dark:border-sky-800',
  }
  return classes[utility.value?.type] || classes.electricity
})

const consumptionUnit = computed(() => {
  const units = { electricity: 'kWh', gas: 'Smc', water: 'mc', waste: '' }
  return units[utility.value?.type] || ''
})

const serviceCodeLabel = computed(() => {
  if (!utility.value?.type) return t('utilities.detail.serviceCode.default')
  const key = `utilities.detail.serviceCode.${utility.value.type}`
  const label = t(key)
  return label === key ? t('utilities.detail.serviceCode.default') : label
})

// ── Functions ──

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

function goBack() {
  router.push('/utilities')
}

async function loadUtility() {
  const id = route.params.id
  loading.value = true
  try {
    const data = await utilitiesStore.fetchUtility(id)
    utility.value = data
  } catch {
    router.push('/utilities')
  } finally {
    loading.value = false
  }
}

async function refreshUtility() {
  try {
    const data = await utilitiesStore.fetchUtility(utility.value.id)
    utility.value = data
  } catch (err) {
    console.error('Error refreshing utility:', err)
  }
}

async function onDataChanged() {
  await refreshUtility()
  analysisTab.value?.refreshComparison()
}

function onThresholdSaved(thresholds) {
  utility.value.comparison_threshold = thresholds.comparison_threshold
  utility.value.threshold_per_day = thresholds.threshold_per_day
}

async function onUtilityUpdated(updatedUtility) {
  showEditModal.value = false
  utility.value = updatedUtility
  window.$toast?.success(t('utilities.detail.updatedToast'))
}

async function confirmDeleteUtility() {
  const ok = await confirm({
    title: t('utilities.detail.deleteConfirm.title'),
    message: t('utilities.detail.deleteConfirm.message'),
    confirmText: t('utilities.detail.deleteConfirm.action'),
    variant: 'danger'
  })
  if (!ok) return
  try {
    await utilitiesStore.deleteUtility(utility.value.id)
    router.push('/utilities')
  } catch (err) {
    console.error('Error deleting utility:', err)
  }
}

// ── Init ──

onMounted(() => {
  loadUtility()
})
</script>
