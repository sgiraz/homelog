<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-3xl font-bold text-ink">
        {{ t('devices.title') }}
      </h1>
      <RouterLink
          to="/devices/new"
          class="px-4 py-2 rounded-lg bg-primary text-white hover:opacity-90"
      >
        {{ t('devices.add') }}
      </RouterLink>
    </div>
    <Card class="p-4 mx-6">
      <div class="space-y-3">
        <!-- Suche -->
        <div class="relative">
          <svg
              class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-faint"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
          >
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0"
            />
          </svg>

          <input
              v-model="search"
              type="search"
              :placeholder="t('devices.searchPlaceholder')"
              class="w-full pl-9 pr-4 py-2 border border-line rounded-lg
                 bg-surface text-ink text-sm
                 focus:outline-none focus:ring-2 focus:ring-blue-500"
              @keyup.enter="searchDevices"
          />
        </div>

        <!-- Filter -->
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex items-center gap-2">
            <label class="text-sm text-ink-soft whitespace-nowrap">
              {{ t('devices.allStatuses') }}
            </label>

            <select
                v-model="status"
                class="px-3 py-2 border border-line rounded-lg
                   bg-surface text-ink text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
                @change="filterDevices"
            >
              <option value="">
                {{ t('devices.allStatuses') }}
              </option>
              <option value="active">
                {{ t('devices.status.active') }}
              </option>
              <option value="broken">
                {{ t('devices.status.broken') }}
              </option>
              <option value="repair">
                {{ t('devices.status.repair') }}
              </option>
              <option value="disposed">
                {{ t('devices.status.disposed') }}
              </option>
              <option value="retired">
                {{ t('devices.status.retired') }}
              </option>
            </select>
          </div>

          <div class="flex items-center gap-2">
            <label class="text-sm text-ink-soft whitespace-nowrap">
              {{ t('devices.allCategories') }}
            </label>

            <select
                v-model.number="categoryId"
                class="px-3 py-2 border border-line rounded-lg
                   bg-surface text-ink text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
                @change="filterDevices"
            >
              <option :value="null">
                {{ t('devices.allCategories') }}
              </option>

              <option
                  v-for="category in devicesStore.categories"
                  :key="category.id"
                  :value="category.id"
              >
                {{ category.icon }}
                {{ category.user_id
                  ? category.name
                  : t(`device_categories.${category.slug}`) }}
              </option>
            </select>
          </div>
        </div>
      </div>
    </Card>

    <div v-if="devicesStore.loading">
      {{ t('devices.loading') }}
    </div>

    <div v-else>
      <div v-if="devicesStore.devices.length === 0">
        {{ t('devices.empty') }}
      </div>

      <div v-else class="p-6 min-w-0">
        <Card
            v-for="device in devicesStore.devices"
            :key="device.id"
            className="p-6 group relative"
            @click="router.push(`/devices/${device.id}`)"

        >
          <div class="flex items-start justify-between mb-4">
            <div class="flex items-center gap-3">
              <div class="text-3xl">{{ device.category?.icon || '🖥️' }}</div>
              <div>
                <h2 class="text-lg font-bold text-ink">
                  {{ device.name }}
                </h2>
                <p v-if="device.manufacturer || device.model" class="text-sm text-ink-soft">
                  {{ device.manufacturer }} {{ device.model }}
                </p>
                <p v-if="device.category?.name" class="text-xs text-ink-faint mt-1">
                  {{ device.category.user_id
                    ? device.category.name
                    : t(`device_categories.${device.category.slug}`) }}
                </p>
              </div>
            </div>

            <div class="flex flex-col items-end gap-1">
              <Badge variant="positive">
                {{ t(`devices.status.${device.status}`) }}
              </Badge>

              <div class="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                    type="button"
                    class="p-1.5 text-blue-600 hover:text-blue-700 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded"
                    :aria-label="t('devices.editAria')"
                    @click.stop="router.push(`/devices/${device.id}/edit`)"
                >
                  ✎
                </button>

                <button
                    type="button"
                    class="p-1.5 text-danger-soft hover:bg-danger/10 rounded"
                    :aria-label="t('devices.deleteAria')"
                    @click.stop="deleteDevice(device.id)"
                >
                  🗑
                </button>
              </div>
            </div>
          </div>

          <p v-if="device.location" class="text-sm text-ink-soft">
            {{ t('devices.card.location') }}: {{ device.location }}
          </p>
          <p v-if="device.property?.name" class="text-sm text-ink-soft">
            {{ t('devices.card.property') }}: {{ device.property.name }}
          </p>
          <p v-if="device.purchase_price != null" class="text-sm text-ink-soft">
            {{ t('devices.card.purchasePrice') }}:
            {{ _formatCurrency(device.purchase_price)}}
          </p>
          <p v-if="device.purchase_date" class="text-sm text-ink-soft">
            {{ t('devices.card.purchaseDate') }}:
            {{ _formatDate(device.purchase_date) }}
          </p>
        </Card>
        <div
            v-if="devicesStore.total > devicesStore.limit"
            class="flex items-center justify-between mt-6"
        >
          <button
              type="button"
              :disabled="devicesStore.offset <= 0"
              class="px-4 py-2 rounded-lg border border-line text-ink
           disabled:opacity-40 disabled:cursor-not-allowed"
              @click="previousPage"
          >
            ←
          </button>

          <span class="text-sm text-ink-soft">
    {{ devicesStore.offset + 1 }}
    –
    {{ Math.min(devicesStore.offset + devicesStore.limit, devicesStore.total) }}
    / {{ devicesStore.total }}
  </span>

          <button
              type="button"
              :disabled="devicesStore.offset + devicesStore.limit >= devicesStore.total"
              class="px-4 py-2 rounded-lg border border-line text-ink
           disabled:opacity-40 disabled:cursor-not-allowed"
              @click="nextPage"
          >
            →
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import {useRouter} from "vue-router";

defineOptions({ name: 'DevicesView' })

import { onMounted, ref } from 'vue'
import { useDevicesStore } from '@/stores/devices'
import { useI18n}  from "vue-i18n";
import { formatDate as _formatDate, formatCurrency as _formatCurrency } from '@/utils/dateFormatter'
import Card from '@/components/common/Card.vue'
import Badge from '@/components/common/Badge.vue'

const devicesStore = useDevicesStore()
const { t } = useI18n()
const router = useRouter()
const search = ref('')
const status = ref('')
const categoryId = ref(null)

async function filterDevices() {
  await devicesStore.fetchDevices({
    q: search.value.trim() || undefined,
    status: status.value || undefined,
    category_id: categoryId.value || undefined,
    limit: devicesStore.limit,
    offset: 0
  })
}
async function previousPage() {
  if (devicesStore.offset <= 0) return

  await devicesStore.fetchDevices({
    q: search.value.trim() || undefined,
    status: status.value || undefined,
    category_id: categoryId.value || undefined,
    limit: devicesStore.limit,
    offset: Math.max(0, devicesStore.offset - devicesStore.limit)
  })
}

async function nextPage() {
  if (devicesStore.offset + devicesStore.limit >= devicesStore.total) return

  await devicesStore.fetchDevices({
    q: search.value.trim() || undefined,
    status: status.value || undefined,
    category_id: categoryId.value || undefined,
    limit: devicesStore.limit,
    offset: devicesStore.offset + devicesStore.limit
  })
}
async function searchDevices() {
  await filterDevices()
}
async function deleteDevice(deviceId) {
  if (!window.confirm(t('devices.deleteConfirm'))) return

  try {
    await devicesStore.deleteDevice(deviceId)
    await searchDevices()
  } catch (error) {
    console.error('Delete device failed:', error)
  }
}
onMounted(async () => {
  try {
    await devicesStore.fetchCategories()
    await searchDevices()
  } catch (error) {
    console.error('Fetch devices failed:', error)
  }
})
</script>