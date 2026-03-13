<template>
  <div class="space-y-4">
    <!-- Back + Header -->
    <div class="flex items-center gap-3">
      <button
        @click="goBack"
        class="p-2 -ml-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        aria-label="Torna ai servizi"
      >
        <svg class="w-5 h-5 text-gray-600 dark:text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>
      <div v-if="utility" class="flex items-center gap-3 flex-1 min-w-0">
        <div :class="['p-2.5 rounded-xl border flex-shrink-0', utilityColorClass]">
          <span class="text-xl">{{ utilityIcon }}</span>
        </div>
        <div class="min-w-0">
          <h1 class="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white truncate">{{ utility.provider }}</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ utilityTypeLabel }}</p>
        </div>
      </div>
      <!-- Actions -->
      <div v-if="utility" class="flex items-center gap-1 flex-shrink-0">
        <a
          v-if="utility.customer_portal"
          :href="utility.customer_portal"
          target="_blank"
          class="p-2.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-400"
          title="Area clienti"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
        </a>
        <button
          v-if="authStore.isAdmin"
          @click="showEditModal = true"
          class="p-2.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-400"
          title="Modifica servizio"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
          </svg>
        </button>
        <button
          v-if="authStore.isAdmin"
          @click="confirmDeleteUtility"
          class="p-2.5 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 text-gray-400 hover:text-red-500 dark:hover:text-red-400"
          title="Elimina servizio"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12 text-gray-500 dark:text-gray-400">
      <div class="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
      Caricamento...
    </div>

    <template v-else-if="utility">
      <!-- Info Cards -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div v-if="utility.service_code" class="p-3 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
          <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">{{ serviceCodeLabel }}</div>
          <div class="text-sm font-medium text-gray-900 dark:text-white truncate">{{ utility.service_code }}</div>
        </div>
        <div v-if="utility.customer_code" class="p-3 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
          <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">{{ isMetered ? 'Cliente' : 'Contratto' }}</div>
          <div class="text-sm font-medium text-gray-900 dark:text-white truncate">{{ utility.customer_code }}</div>
        </div>
        <div v-if="utility.power_capacity" class="p-3 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
          <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Potenza</div>
          <div class="text-sm font-medium text-gray-900 dark:text-white">{{ utility.power_capacity }} kW</div>
        </div>
        <div v-if="utility.recurring_amount" class="p-3 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
          <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Canone</div>
          <div class="text-sm font-medium text-gray-900 dark:text-white">{{ formatCurrency(utility.recurring_amount) }}/{{ billingFrequencyLabel }}</div>
        </div>
        <div v-if="utility.paid_by_member?.name" class="p-3 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
          <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Pagato da</div>
          <div class="text-sm font-medium text-gray-900 dark:text-white truncate">{{ utility.paid_by_member.name }}</div>
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
                : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
            ]"
          >
            <span>{{ tab.icon }}</span>
            <span>{{ tab.label }}</span>
            <span v-if="tab.count != null" class="text-xs opacity-70">({{ tab.count }})</span>
          </button>
        </div>
      </div>

      <!-- ═══ Bollette Tab ═══ -->
      <div v-show="activeTab === 'bills'">
        <!-- Filters -->
        <div class="flex flex-col sm:flex-row gap-2 mb-4">
          <div class="flex gap-2 flex-1">
            <input
              v-model="billSearch"
              type="search"
              placeholder="Cerca bolletta..."
              class="flex-1 px-3 py-2.5 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <select
              v-model="billStatusFilter"
              class="px-3 py-2.5 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">Tutte</option>
              <option value="unpaid">Da pagare</option>
              <option value="paid">Pagate</option>
            </select>
          </div>
          <div class="flex gap-2">
            <input
              v-model="billDateFrom"
              type="date"
              class="px-3 py-2.5 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
              title="Da data"
            />
            <input
              v-model="billDateTo"
              type="date"
              class="px-3 py-2.5 border border-gray-200 dark:border-gray-700 rounded-lg
                     bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500"
              title="A data"
            />
          </div>
        </div>

        <!-- Bill Summary -->
        <div v-if="filteredBills.length > 0" class="mb-3 flex items-center justify-between">
          <span class="text-sm text-gray-500 dark:text-gray-400">
            {{ filteredBills.length }} bollette — Totale: {{ formatCurrency(filteredBillsTotal) }}
          </span>
          <Button size="sm" @click="openAddBill">
            <svg class="w-4 h-4 sm:mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            <span class="hidden sm:inline">Aggiungi</span>
          </Button>
        </div>

        <!-- Empty -->
        <div v-if="filteredBills.length === 0" class="text-center py-8">
          <p class="text-gray-500 dark:text-gray-400 mb-3">
            {{ utility.bills?.length ? 'Nessuna bolletta corrisponde ai filtri' : 'Nessuna bolletta registrata' }}
          </p>
          <Button size="sm" @click="openAddBill">Aggiungi bolletta</Button>
        </div>

        <!-- Bills List -->
        <div v-else class="space-y-2">
          <div
            v-for="bill in filteredBills"
            :key="bill.id"
            class="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl
                   hover:border-gray-300 dark:hover:border-gray-600 transition-colors"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-semibold text-gray-900 dark:text-white">
                    {{ formatCurrency(bill.amount_total) }}
                  </span>
                  <span :class="[
                    'px-2 py-0.5 text-xs rounded-full font-medium',
                    bill.is_paid
                      ? 'bg-green-100 dark:bg-green-900/50 text-green-700 dark:text-green-300'
                      : isDueSoon(bill)
                        ? 'bg-yellow-100 dark:bg-yellow-900/50 text-yellow-700 dark:text-yellow-300'
                        : 'bg-red-100 dark:bg-red-900/50 text-red-700 dark:text-red-300'
                  ]">
                    {{ bill.is_paid ? 'Pagata' : isDueSoon(bill) ? 'In scadenza' : 'Da pagare' }}
                  </span>
                </div>
                <div class="text-sm text-gray-500 dark:text-gray-400 mt-1">
                  {{ formatPeriod(bill.period_start, bill.period_end) }}
                </div>
                <div class="flex items-center gap-3 text-xs text-gray-400 dark:text-gray-500 mt-1">
                  <span>Scad. {{ formatDate(bill.due_date) }}</span>
                  <span v-if="bill.consumption_total">{{ formatConsumption(bill.consumption_total) }} {{ consumptionUnit }}</span>
                  <span v-if="bill.bill_number" class="font-mono">N. {{ bill.bill_number }}</span>
                </div>
              </div>

              <!-- Actions -->
              <div class="flex items-center gap-1 flex-shrink-0">
                <button
                  v-if="!bill.is_paid"
                  @click="markBillAsPaid(bill)"
                  class="p-2.5 rounded-lg text-green-600 hover:bg-green-50 dark:hover:bg-green-900/20"
                  title="Segna come pagata"
                >
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                  </svg>
                </button>
                <button
                  @click="openEditBill(bill)"
                  class="p-2.5 rounded-lg text-gray-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20"
                  title="Modifica"
                >
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                  </svg>
                </button>
                <button
                  @click="confirmDeleteBill(bill)"
                  class="p-2.5 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
                  title="Elimina"
                >
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- ═══ Letture Tab ═══ -->
      <div v-show="activeTab === 'readings'">
        <div class="flex justify-between items-center mb-4">
          <span class="text-sm text-gray-500 dark:text-gray-400">
            {{ utility.readings?.length || 0 }} letture registrate
          </span>
          <Button size="sm" @click="openAddReading">
            <svg class="w-4 h-4 sm:mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            <span class="hidden sm:inline">Aggiungi</span>
          </Button>
        </div>

        <div v-if="!utility.readings?.length" class="text-center py-8">
          <p class="text-gray-500 dark:text-gray-400 mb-3">Nessuna lettura registrata</p>
          <Button size="sm" @click="openAddReading">Aggiungi lettura</Button>
        </div>

        <!-- Timeline -->
        <div v-else class="space-y-1">
          <div
            v-for="(reading, idx) in utility.readings"
            :key="reading.id"
            class="flex gap-3"
          >
            <!-- Timeline line -->
            <div class="flex flex-col items-center w-6 flex-shrink-0">
              <div class="w-3 h-3 rounded-full mt-4"
                :class="idx === 0 ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'"
              />
              <div v-if="idx < utility.readings.length - 1" class="w-px flex-1 bg-gray-200 dark:bg-gray-700" />
            </div>
            <!-- Content -->
            <div class="flex-1 pb-4 min-w-0">
              <div class="p-3 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
                <div class="flex items-start justify-between gap-2">
                  <div class="min-w-0">
                    <div class="font-medium text-gray-900 dark:text-white">
                      <template v-if="utility.type === 'electricity'">
                        <span v-if="reading.value_f1" class="mr-2">F1: {{ reading.value_f1 }}</span>
                        <span v-if="reading.value_f2" class="mr-2">F2: {{ reading.value_f2 }}</span>
                        <span v-if="reading.value_f3">F3: {{ reading.value_f3 }}</span>
                        <span class="text-gray-400 text-sm ml-1">kWh</span>
                      </template>
                      <template v-else>
                        {{ reading.value || '-' }} {{ consumptionUnit }}
                      </template>
                    </div>
                    <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400 mt-0.5 flex-wrap">
                      <span>{{ formatDate(reading.reading_date) }}</span>
                      <span v-if="reading.source === 'submitted'" class="px-1.5 py-0.5 bg-blue-100 dark:bg-blue-900/50 text-blue-600 dark:text-blue-300 text-xs rounded">
                        Inviata
                      </span>
                      <span v-if="readingBillMap[reading.id]" class="px-1.5 py-0.5 bg-green-100 dark:bg-green-900/50 text-green-600 dark:text-green-300 text-xs rounded font-mono">
                        Boll. {{ readingBillMap[reading.id] }}
                      </span>
                    </div>
                    <div v-if="reading.notes" class="text-xs text-gray-400 mt-1">{{ reading.notes }}</div>
                  </div>
                  <div class="flex items-center gap-0.5 flex-shrink-0">
                    <button
                      @click="openEditReading(reading)"
                      class="p-2 rounded-lg text-gray-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20"
                      title="Modifica"
                    >
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </button>
                    <button
                      @click="confirmDeleteReading(reading)"
                      class="p-2 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
                      title="Elimina"
                    >
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- ═══ Analisi Tab ═══ -->
      <div v-show="activeTab === 'analysis'" class="space-y-4">
        <!-- Period Filter for Analysis -->
        <Card class="p-4">
          <div class="flex flex-col sm:flex-row gap-3 items-start sm:items-center">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">Periodo:</span>
            <div class="flex gap-2 flex-wrap">
              <button
                v-for="preset in periodPresets"
                :key="preset.id"
                @click="setAnalysisPeriod(preset)"
                :class="[
                  'px-3 py-2 text-sm rounded-lg transition-colors',
                  analysisPeriod === preset.id
                    ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 font-medium'
                    : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
                ]"
              >
                {{ preset.label }}
              </button>
            </div>
          </div>
          <div v-if="analysisPeriod === 'custom'" class="flex gap-2 mt-3">
            <input v-model="analysisFrom" type="date" class="px-3 py-2.5 border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
            <input v-model="analysisTo" type="date" class="px-3 py-2.5 border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
        </Card>

        <!-- Analysis KPIs -->
        <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
          <Card class="p-4 text-center">
            <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ formatCurrency(analysisData.totalSpent) }}</div>
            <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">Spesa totale</div>
          </Card>
          <Card class="p-4 text-center">
            <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ formatConsumption(analysisData.totalConsumption) }}</div>
            <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">Consumo ({{ consumptionUnit }})</div>
          </Card>
          <Card class="p-4 text-center col-span-2 sm:col-span-1">
            <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ analysisData.billCount }}</div>
            <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">Bollette nel periodo</div>
          </Card>
        </div>

        <!-- Comparison Section -->
        <div v-if="utility.type !== 'waste'" class="space-y-4">
          <!-- Threshold Settings (collapsible) -->
          <Card class="p-4">
            <button
              @click="showThresholdSettings = !showThresholdSettings"
              class="flex items-center justify-between w-full text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              <span>Impostazioni soglia confronto</span>
              <svg :class="['w-4 h-4 transition-transform', showThresholdSettings ? 'rotate-180' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>

            <div v-if="showThresholdSettings" class="mt-3 space-y-3">
              <div class="flex items-center justify-between">
                <div>
                  <div class="text-sm text-gray-600 dark:text-gray-400">Soglia base</div>
                  <div class="text-xs text-gray-400">Stesso giorno</div>
                </div>
                <div class="flex items-center gap-2">
                  <input v-model.number="thresholdValue" type="number" min="0.5" max="50" step="0.5"
                    class="w-16 px-2 py-2 text-sm text-center border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-blue-500" />
                  <span class="text-xs text-gray-400">{{ consumptionUnit }}</span>
                </div>
              </div>
              <div class="flex items-center justify-between">
                <div>
                  <div class="text-sm text-gray-600 dark:text-gray-400">Per giorno</div>
                  <div class="text-xs text-gray-400">Tolleranza aggiuntiva</div>
                </div>
                <div class="flex items-center gap-2">
                  <input v-model.number="thresholdPerDayValue" type="number" min="0.1" max="10" step="0.1"
                    class="w-16 px-2 py-2 text-sm text-center border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-1 focus:ring-blue-500" />
                  <span class="text-xs text-gray-400">{{ consumptionUnit }}/g</span>
                </div>
              </div>
              <Button v-if="hasThresholdChanges" size="sm" @click="saveThreshold" :disabled="savingThreshold">
                {{ savingThreshold ? 'Salvataggio...' : 'Salva soglia' }}
              </Button>
            </div>
          </Card>

          <!-- Comparison Card -->
          <ReadingComparisonCard
            ref="comparisonCard"
            :key="comparisonKey"
            :utility-id="utility.id"
            :utility-type="utility.type"
            :base-threshold="utility.comparison_threshold || 2"
            :threshold-per-day="utility.threshold_per_day || 1"
          />
        </div>
      </div>
      <!-- ═══ Storico Prezzi Tab (fixed services) ═══ -->
      <div v-show="activeTab === 'price_history'" class="space-y-4">
        <div class="flex justify-between items-center">
          <span class="text-sm text-gray-500 dark:text-gray-400">
            {{ utility.price_changes?.length || 0 }} variazioni registrate
          </span>
        </div>

        <div v-if="!utility.price_changes?.length" class="text-center py-8">
          <p class="text-gray-500 dark:text-gray-400">Nessuna variazione di prezzo registrata</p>
          <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">Le variazioni verranno rilevate automaticamente dalle fatture</p>
        </div>

        <!-- Price history timeline -->
        <div v-else class="space-y-1">
          <div
            v-for="(change, idx) in utility.price_changes"
            :key="change.id"
            class="flex gap-3"
          >
            <div class="flex flex-col items-center w-6 flex-shrink-0">
              <div class="w-3 h-3 rounded-full mt-4"
                :class="change.new_amount > change.old_amount ? 'bg-red-500' : 'bg-green-500'"
              />
              <div v-if="idx < utility.price_changes.length - 1" class="w-px flex-1 bg-gray-200 dark:bg-gray-700" />
            </div>
            <div class="flex-1 pb-4 min-w-0">
              <div class="p-3 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
                <div class="flex items-center justify-between gap-2">
                  <div class="min-w-0">
                    <div class="font-medium text-gray-900 dark:text-white">
                      {{ formatCurrency(change.old_amount) }} → {{ formatCurrency(change.new_amount) }}
                      <span :class="change.new_amount > change.old_amount ? 'text-red-500' : 'text-green-500'" class="text-sm ml-1">
                        ({{ change.new_amount > change.old_amount ? '+' : '' }}{{ formatCurrency(change.new_amount - change.old_amount) }})
                      </span>
                    </div>
                    <div class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
                      Dal {{ formatDate(change.effective_date) }}
                    </div>
                    <div v-if="change.reason" class="text-xs text-gray-400 mt-1">{{ change.reason }}</div>
                    <div v-if="change.cancellation_deadline" class="mt-1 px-2 py-1 bg-yellow-50 dark:bg-yellow-900/20 rounded text-xs text-yellow-700 dark:text-yellow-300 inline-block">
                      Recesso entro il {{ formatDate(change.cancellation_deadline) }}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Current price summary -->
        <Card v-if="utility.recurring_amount" class="p-4">
          <div class="text-center">
            <div class="text-xs text-gray-500 dark:text-gray-400 mb-1">Importo attuale</div>
            <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ formatCurrency(utility.recurring_amount) }}</div>
            <div class="text-xs text-gray-400 mt-1">al mese</div>
          </div>
        </Card>
      </div>

    </template>

    <!-- Modals -->
    <AddBillModal
      v-if="showBillModal"
      :utility="utility"
      :bill="editingBill"
      @close="closeBillModal"
      @saved="onBillSaved"
    />

    <AddReadingModal
      v-if="showReadingModal"
      :utility="utility"
      :reading="editingReading"
      @close="closeReadingModal"
      @saved="onReadingSaved"
    />

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
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUtilitiesStore } from '@/stores/utilities'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { utilitiesAPI } from '@/api/client'
import { formatDate as _formatDate, formatPeriod as _formatPeriod, formatNumber as _formatNumber, formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import AddBillModal from '@/components/utilities/AddBillModal.vue'
import AddReadingModal from '@/components/utilities/AddReadingModal.vue'
import EditUtilityModal from '@/components/utilities/EditUtilityModal.vue'
import ReadingComparisonCard from '@/components/utilities/ReadingComparisonCard.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const utilitiesStore = useUtilitiesStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

const loading = ref(true)
const utility = ref(null)
const activeTab = ref('bills')

// Bill filters
const billSearch = ref('')
const billStatusFilter = ref('all')
const billDateFrom = ref('')
const billDateTo = ref('')

// Reading bill map
const readingBillMap = ref({})

// Edit utility modal
const showEditModal = ref(false)

// Bill modal
const showBillModal = ref(false)
const editingBill = ref(null)

// Reading modal
const showReadingModal = ref(false)
const editingReading = ref(null)

// Comparison
const comparisonCard = ref(null)
const comparisonKey = ref(0)
const showThresholdSettings = ref(false)
const thresholdValue = ref(2)
const thresholdPerDayValue = ref(1)
const savingThreshold = ref(false)

// Analysis
const analysisPeriod = ref('12m')
const analysisFrom = ref('')
const analysisTo = ref('')

const periodPresets = [
  { id: '3m', label: '3 mesi', months: 3 },
  { id: '6m', label: '6 mesi', months: 6 },
  { id: '12m', label: '1 anno', months: 12 },
  { id: 'all', label: 'Tutto', months: 0 },
  { id: 'custom', label: 'Personalizzato', months: 0 },
]

// ── Computed ──

const isMetered = computed(() => {
  return ['electricity', 'gas', 'water', 'waste'].includes(utility.value?.type)
})

const billingFrequencyLabel = computed(() => {
  const n = utility.value?.billing_interval || 1
  const u = utility.value?.billing_unit || 'month'
  const labels = { day: ['giorno', 'giorni'], week: ['settimana', 'settimane'], month: ['mese', 'mesi'], year: ['anno', 'anni'] }
  const [singular, plural] = labels[u] || labels.month
  return n === 1 ? singular : `${n} ${plural}`
})

const tabs = computed(() => {
  const billLabel = isMetered.value ? 'Bollette' : 'Fatture'
  const t = [
    { id: 'bills', label: billLabel, icon: '\uD83D\uDCC4', count: utility.value?.bills?.length || 0 },
  ]
  if (isMetered.value) {
    t.push({ id: 'readings', label: 'Letture', icon: '\uD83D\uDCCA', count: utility.value?.readings?.length || 0 })
    if (utility.value?.type !== 'waste') {
      t.push({ id: 'analysis', label: 'Analisi', icon: '\uD83D\uDCC8', count: null })
    }
  } else {
    t.push({ id: 'price_history', label: 'Storico Prezzi', icon: '\uD83D\uDCC8', count: utility.value?.price_changes?.length || 0 })
  }
  return t
})

const utilityIcon = computed(() => {
  const icons = {
    electricity: '\u26A1', gas: '\uD83D\uDD25', water: '\uD83D\uDCA7', waste: '\u267B\uFE0F',
    internet: '\uD83C\uDF10', insurance: '\uD83D\uDEE1\uFE0F', affitto: '\uD83C\uDFE0', mutuo: '\uD83C\uDFE6'
  }
  return icons[utility.value?.type] || '\u26A1'
})

const utilityTypeLabel = computed(() => {
  const labels = {
    electricity: 'Luce', gas: 'Gas', water: 'Acqua', waste: 'Rifiuti',
    internet: 'Internet', insurance: 'Assicurazione', affitto: 'Affitto', mutuo: 'Mutuo'
  }
  return labels[utility.value?.type] || ''
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
  const labels = { electricity: 'POD', gas: 'PDR', internet: 'Numero linea', affitto: 'Rif. Contratto', mutuo: 'N. Mutuo' }
  return labels[utility.value?.type] || 'Codice'
})

const filteredBills = computed(() => {
  let bills = utility.value?.bills || []

  if (billStatusFilter.value === 'paid') bills = bills.filter(b => b.is_paid)
  else if (billStatusFilter.value === 'unpaid') bills = bills.filter(b => !b.is_paid)

  if (billSearch.value.trim()) {
    const q = billSearch.value.toLowerCase()
    bills = bills.filter(b =>
      b.bill_number?.toLowerCase().includes(q) ||
      String(b.amount_total).includes(q) ||
      b.provider?.toLowerCase().includes(q)
    )
  }

  if (billDateFrom.value) {
    const from = new Date(billDateFrom.value)
    bills = bills.filter(b => new Date(b.period_end) >= from)
  }
  if (billDateTo.value) {
    const to = new Date(billDateTo.value)
    bills = bills.filter(b => new Date(b.period_start) <= to)
  }

  return bills
})

const filteredBillsTotal = computed(() => {
  return filteredBills.value.reduce((sum, b) => sum + (b.amount_total || 0), 0)
})

const hasThresholdChanges = computed(() => {
  return thresholdValue.value !== (utility.value?.comparison_threshold || 2) ||
         thresholdPerDayValue.value !== (utility.value?.threshold_per_day || 1)
})

const analysisData = computed(() => {
  const bills = utility.value?.bills || []
  let filtered = bills

  if (analysisPeriod.value !== 'all') {
    let from, to
    if (analysisPeriod.value === 'custom') {
      from = analysisFrom.value ? new Date(analysisFrom.value) : null
      to = analysisTo.value ? new Date(analysisTo.value) : null
    } else {
      const preset = periodPresets.find(p => p.id === analysisPeriod.value)
      if (preset?.months) {
        to = new Date()
        from = new Date()
        from.setMonth(from.getMonth() - preset.months)
      }
    }
    if (from) filtered = filtered.filter(b => new Date(b.period_end) >= from)
    if (to) filtered = filtered.filter(b => new Date(b.period_start) <= to)
  }

  return {
    totalSpent: filtered.reduce((s, b) => s + (b.amount_total || 0), 0),
    totalConsumption: filtered.reduce((s, b) => s + (b.consumption_total || 0), 0),
    billCount: filtered.length,
  }
})

// ── Functions ──

function formatCurrency(value) {
  return _formatCurrency(value, settingsStore.formatSettings)
}

function formatConsumption(value) {
  if (value == null || value === 0) return '0'
  return _formatNumber(parseFloat(value), settingsStore.formatSettings)
}

function formatDate(dateStr) {
  return _formatDate(dateStr, settingsStore.dateSettings)
}

function formatPeriod(start, end) {
  return _formatPeriod(start, end, settingsStore.dateSettings)
}

function isDueSoon(bill) {
  const now = new Date()
  const dueDate = new Date(bill.due_date)
  const threeDays = new Date(now.getTime() + 3 * 24 * 60 * 60 * 1000)
  return dueDate <= threeDays && dueDate >= now
}

function setAnalysisPeriod(preset) {
  analysisPeriod.value = preset.id
}

function goBack() {
  router.push('/utilities')
}

// ── Data Loading ──

async function loadUtility() {
  const id = route.params.id
  loading.value = true
  try {
    const data = await utilitiesStore.fetchUtility(id)
    utility.value = data
    thresholdValue.value = data.comparison_threshold || 2
    thresholdPerDayValue.value = data.threshold_per_day || 1
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
    await fetchReadingBillMap()
  } catch (err) {
    console.error('Error refreshing utility:', err)
  }
}


async function fetchReadingBillMap() {
  if (!utility.value) return
  try {
    const { data } = await utilitiesAPI.getReadings(utility.value.id)
    const map = {}
    for (const r of data || []) {
      if (r.associated_bill_number) map[r.id] = r.associated_bill_number
    }
    readingBillMap.value = map
  } catch { /* non-critical */ }
}

// ── Bill Operations ──

function openAddBill() {
  editingBill.value = null
  showBillModal.value = true
}

function openEditBill(bill) {
  editingBill.value = bill
  showBillModal.value = true
}

function closeBillModal() {
  showBillModal.value = false
  editingBill.value = null
}

async function onBillSaved() {
  closeBillModal()
  await refreshUtility()
  comparisonKey.value++
}

async function markBillAsPaid(bill) {
  try {
    await utilitiesStore.updateBill(utility.value.id, bill.id, {
      is_paid: true,
      paid_date: new Date().toISOString()
    })
    await refreshUtility()
  } catch (err) {
    console.error('Error marking bill as paid:', err)
  }
}

async function confirmDeleteBill(bill) {
  const ok = await confirm({
    title: 'Elimina bolletta',
    message: `Eliminare la bolletta di ${formatCurrency(bill.amount_total)}?`,
    confirmText: 'Elimina',
    variant: 'danger'
  })
  if (!ok) return
  try {
    await utilitiesStore.deleteBill(utility.value.id, bill.id)
    await refreshUtility()
    comparisonKey.value++
  } catch (err) {
    console.error('Error deleting bill:', err)
  }
}

// ── Reading Operations ──

function openAddReading() {
  editingReading.value = null
  showReadingModal.value = true
}

function openEditReading(reading) {
  editingReading.value = reading
  showReadingModal.value = true
}

function closeReadingModal() {
  showReadingModal.value = false
  editingReading.value = null
}

async function onReadingSaved() {
  closeReadingModal()
  await refreshUtility()
  comparisonKey.value++
}

async function confirmDeleteReading(reading) {
  const ok = await confirm({
    title: 'Elimina lettura',
    message: 'Sei sicuro di voler eliminare questa lettura?',
    confirmText: 'Elimina',
    variant: 'danger'
  })
  if (!ok) return
  try {
    await utilitiesStore.deleteReading(utility.value.id, reading.id)
    await refreshUtility()
    comparisonKey.value++
  } catch (err) {
    console.error('Error deleting reading:', err)
  }
}

// ── Utility Operations ──

async function onUtilityUpdated(updatedUtility) {
  showEditModal.value = false
  utility.value = updatedUtility
  window.$toast?.success('Servizio aggiornato')
}

async function confirmDeleteUtility() {
  const ok = await confirm({
    title: 'Elimina servizio',
    message: 'Eliminare questo servizio e tutti i dati associati (bollette, letture)?',
    confirmText: 'Elimina',
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

async function saveThreshold() {
  savingThreshold.value = true
  try {
    await utilitiesStore.updateUtility(utility.value.id, {
      comparison_threshold: thresholdValue.value,
      threshold_per_day: thresholdPerDayValue.value
    })
    utility.value.comparison_threshold = thresholdValue.value
    utility.value.threshold_per_day = thresholdPerDayValue.value
    comparisonKey.value++
  } catch (err) {
    console.error('Error saving threshold:', err)
  } finally {
    savingThreshold.value = false
  }
}

// ── Watchers ──

watch(activeTab, (tab) => {
  if (tab === 'readings') fetchReadingBillMap()
})

// ── Init ──

onMounted(() => {
  loadUtility()
})
</script>
