<template>
  <div class="space-y-6">
    <h1 class="text-3xl font-bold text-ink">
      {{ device?.name || t('devices.detail.edit') }}
    </h1>

    <Card v-if="device" className="p-6">
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.name') }}
          </label>
          <input
              v-model="device.name"
              type="text"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.manufacturer') }}
          </label>
          <input
              v-model="device.manufacturer"
              type="text"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>
        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.model') }}
          </label>
          <input
              v-model="device.model"
              type="text"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.serialNumber') }}
          </label>
          <input
              v-model="device.serial_number"
              type="text"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>
        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.form.property') }}
          </label>

          <select
              v-model="device.property_id"
              class="w-full rounded-lg border border-line bg-surface px-3 py-2 text-ink"
              required
          >
            <option :value="null">
              —
            </option>

            <option
                v-for="property in properties"
                :key="property.id"
                :value="property.id"
            >
              {{ property.name }}
            </option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.category') }}
          </label>

          <select
              v-model="device.category_id"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          >
            <option :value="null">
              —
            </option>

            <option
                v-for="category in categories"
                :key="category.id"
                :value="category.id"
            >
              {{ category.icon }} {{ category.user_id ? category.name : $t(`device_categories.${category.slug}`) }}
            </option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.status') }}
          </label>

          <select
              v-model="device.status"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          >
            <option value="active">{{ t('devices.status.active') }}</option>
            <option value="repair">{{ t('devices.status.repair') }}</option>
            <option value="broken">{{ t('devices.status.broken') }}</option>
            <option value="retired">{{ t('devices.status.retired') }}</option>
            <option value="disposed">{{ t('devices.status.disposed') }}</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.location') }}
          </label>
          <input
              v-model="device.location"
              type="text"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.hostname') }}
          </label>
          <input
              v-model="device.hostname"
              type="text"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.ipAddress') }}
          </label>
          <input
              v-model="device.ip_address"
              type="text"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.macAddress') }}
          </label>
          <input
              v-model="device.mac_address"
              type="text"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.firmware') }}
          </label>
          <input
              v-model="device.firmware"
              type="text"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>
        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.managementUrl') }}
          </label>
          <input
              v-model="device.management_url"
              type="url"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>
        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.purchaseDate') }}
          </label>
          <input
              v-model="device.purchase_date"
              type="date"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.purchasePrice') }}
          </label>
          <input
              v-model="device.purchase_price"
              type="number"
              step="0.01"
              min="0"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>
        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.warrantyUntil') }}
          </label>
          <input
              v-model="device.warranty_until"
              type="date"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-ink mb-1">
            {{ t('devices.detail.notes') }}
          </label>
          <textarea
              v-model="device.notes"
              rows="5"
              class="w-full px-3 py-2 rounded-lg border border-line bg-surface text-ink"
          ></textarea>
        </div>
      </div>
    </Card>

    <div class="flex justify-end">
      <button
          type="button"
          class="px-4 py-2 rounded-lg bg-primary text-white hover:opacity-90 transition-opacity disabled:opacity-50"
          :disabled="saving"
          @click="saveDevice"
      >
        {{ saving ? t('devices.detail.saving') : t('devices.detail.save') }}
      </button>
      <button
          type="button"
          class="px-4 py-2 rounded-lg text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
          @click="deleteDevice"
      >
        {{ t('devices.detail.delete') }}
      </button>
    </div>

  </div>
</template>


<script setup>
defineOptions({ name: 'DeviceEditView' })

import { useI18n } from 'vue-i18n'
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useDevicesStore } from '@/stores/devices'
import Card from '@/components/common/Card.vue'
import { propertiesAPI } from '@/api/client'

const { t } = useI18n()

const route = useRoute()
const devicesStore = useDevicesStore()

const device = ref(null)
const categories = ref([])
const properties = ref([])
const saving = ref(false)
const router = useRouter()

onMounted(async () => {
  if (route.name === 'device-new') {
    device.value = {
      property_id: null,
      category_id: null,
      name: '',
      manufacturer: '',
      model: '',
      serial_number: '',
      status: 'active',
      location: '',
      hostname: '',
      ip_address: '',
      mac_address: '',
      firmware: '',
      management_url: '',
      purchase_date: '',
      purchase_price: null,
      warranty_until: '',
      notes: '',
    }
  } else {
    device.value = await devicesStore.fetchDevice(route.params.id)

    if (device.value?.purchase_date) {
      device.value.purchase_date = device.value.purchase_date.slice(0, 10)
    }

    if (device.value?.warranty_until) {
      device.value.warranty_until = device.value.warranty_until.slice(0, 10)
    }
  }

  categories.value = await devicesStore.fetchCategories()
  const { data } = await propertiesAPI.list()
  properties.value = data
})

async function saveDevice() {
  saving.value = true

  try {
    if (route.name === 'device-new') {
      const createdDevice = await devicesStore.createDevice(device.value)
      router.push(`/devices/${createdDevice.id}`)
    } else {
      await devicesStore.updateDevice(route.params.id, device.value)
      router.push(`/devices/${route.params.id}`)
    }
  } finally {
    saving.value = false
  }
}
async function deleteDevice() {
  if (!confirm(t('devices.detail.deleteConfirm'))) {
    return
  }

  await devicesStore.deleteDevice(route.params.id)
  router.push('/devices')
}

</script>