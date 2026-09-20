<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold text-ink">
          {{ device?.name || t('devices.detail.title') }}
        </h1>

        <p v-if="device?.category?.name" class="text-sm text-ink-faint mt-1">
          {{ device.category.icon || '🖥️' }}
          {{ device.category.name }}
        </p>
      </div>

      <button
          type="button"
          class="px-4 py-2 rounded-lg bg-primary text-white hover:opacity-90 transition-opacity"
          @click="router.push(`/devices/${device.id}/edit`)"
      >
        {{ t('devices.detail.edit') }}
      </button>
    </div>


    <Card v-if="device" className="p-6">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.manufacturer') }}
          </p>
          <p class="text-ink">
            {{ device.manufacturer || '—' }}
          </p>
        </div>

        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.model') }}
          </p>
          <p class="text-ink">
            {{ device.model || '—' }}
          </p>
        </div>
      </div>
    </Card>

    <!-- Card for Network -->
    <Card v-if="device" className="p-6">
      <h2 class="text-lg font-bold text-ink mb-4">
        {{ t('devices.detail.network') }}
      </h2>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.hostname') }}
          </p>
          <p class="text-ink">
            {{ device.hostname || '—' }}
          </p>
        </div>

        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.ipAddress') }}
          </p>
          <p class="text-ink">
            {{ device.ip_address || '—' }}
          </p>
        </div>

        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.macAddress') }}
          </p>
          <p class="text-ink">
            {{ device.mac_address || '—' }}
          </p>
        </div>

        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.firmware') }}
          </p>
          <p class="text-ink">
            {{ device.firmware || '—' }}
          </p>
        </div>
        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.managementUrl') }}
          </p>
          <a
              v-if="device.management_url"
              :href="device.management_url"
              target="_blank"
              rel="noopener noreferrer"
              class="text-primary hover:underline"
          >
            {{ device.management_url }}
          </a>
          <p v-else class="text-ink">
            —
          </p>
        </div>
      </div>
    </Card>

    <!-- Card for General -->
    <Card v-if="device" className="p-6">
      <h2 class="text-lg font-bold text-ink mb-4">
        {{ t('devices.detail.general') }}
      </h2>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.status') }}
          </p>
          <p class="text-ink">
            {{ t(`devices.status.${device.status}`) }}
          </p>
        </div>

        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.location') }}
          </p>
          <p class="text-ink">
            {{ device.location || '—' }}
          </p>
        </div>

        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.serialNumber') }}
          </p>
          <p class="text-ink">
            {{ device.serial_number || '—' }}
          </p>
        </div>
      </div>
    </Card>

    <!-- Card for Guarantee -->
    <Card v-if="device" className="p-6">
      <h2 class="text-lg font-bold text-ink mb-4">
        {{ t('devices.detail.purchase') }}
      </h2>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.purchaseDate') }}
          </p>
          <p class="text-ink">
            {{ device.purchase_date || '—' }}
          </p>
        </div>

        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.purchasePrice') }}
          </p>
          <p class="text-ink">
            {{ device.purchase_price ?? '—' }}
          </p>
        </div>

        <div>
          <p class="text-sm text-ink-faint">
            {{ t('devices.detail.warrantyUntil') }}
          </p>
          <p class="text-ink">
            {{ device.warranty_until || '—' }}
          </p>
        </div>
      </div>
    </Card>
    <!-- Card for Expenses -->
    <Card v-if="device" className="p-6">
      <h2 class="text-lg font-bold text-ink mb-4">
        {{ t('devices.detail.expenses') }}
      </h2>

      <div v-if="device.expenses?.length" class="space-y-3">
        <div
            v-for="expense in device.expenses"
            :key="expense.id"
            class="flex items-center justify-between border-b border-line pb-3 last:border-b-0"
        >
          <div>
            <p class="text-ink font-medium">
              {{ expense.description || '—' }}
            </p>
            <p class="text-sm text-ink-faint">
              {{ expense.date }}
            </p>
          </div>

          <p class="text-ink font-medium">
            {{ Number(expense.amount).toFixed(2) }} €
          </p>
        </div>
      </div>

      <p v-else class="text-ink-faint">
        {{ t('devices.detail.noExpenses') }}
      </p>
    </Card>
    <!-- Card for Notes -->
    <Card v-if="device" className="p-6">
      <h2 class="text-lg font-bold text-ink mb-4">
        {{ t('devices.detail.notes') }}
      </h2>

      <p class="text-ink whitespace-pre-wrap">
        {{ device.notes || '—' }}
      </p>
    </Card>

  </div>
</template>

<script setup>
defineOptions({ name: 'DeviceDetailView' })

import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useDevicesStore } from '@/stores/devices'
import Card from '@/components/common/Card.vue'


const route = useRoute()
const { t } = useI18n()
const devicesStore = useDevicesStore()
const device = ref(null)
const router = useRouter()

onMounted(async () => {
  device.value = await devicesStore.fetchDevice(route.params.id)
})
</script>