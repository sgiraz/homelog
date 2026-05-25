<template>
  <div class="space-y-4">
    <Card class="p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h2 class="text-xl font-bold text-ink">{{ t('settings.properties.title') }}</h2>
          <p class="text-sm text-ink-soft mt-1">{{ t('settings.properties.subtitle') }}</p>
        </div>
        <Button @click="showAddPropertyForm = !showAddPropertyForm" :variant="showAddPropertyForm ? 'secondary' : 'primary'" size="sm">
          <svg v-if="!showAddPropertyForm" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </Button>
      </div>

      <div v-if="allProperties.length > 0" class="space-y-2 mb-4">
        <div
          v-for="prop in allProperties"
          :key="prop.id"
          class="flex items-center gap-3 p-3 rounded-xl border border-line bg-surface"
        >
          <span class="text-2xl">🏠</span>
          <div class="flex-1 min-w-0">
            <div class="font-medium text-ink">{{ prop.name }}</div>
            <div v-if="prop.address" class="text-xs text-ink-muted truncate">{{ prop.address }}</div>
          </div>
          <span v-if="prop.is_current" class="px-2 py-0.5 text-xs rounded-full bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300 font-medium">{{ t('settings.properties.principalBadge') }}</span>
        </div>
      </div>

      <div v-if="showAddPropertyForm" class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-xl border border-blue-200 dark:border-blue-800 space-y-3">
        <h3 class="text-sm font-medium text-ink">{{ t('settings.properties.newTitle') }}</h3>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <input
            v-model="newProperty.name"
            type="text"
            :placeholder="t('settings.properties.namePlaceholder')"
            class="px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <input
            v-model="newProperty.address"
            type="text"
            :placeholder="t('settings.properties.addressPlaceholder')"
            autocomplete="street-address"
            class="px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <select
            v-model="newProperty.type"
            class="px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="owned">{{ t('settings.properties.typeOwned') }}</option>
            <option value="rented">{{ t('settings.properties.typeRented') }}</option>
          </select>
          <div class="flex items-center gap-2">
            <input
              type="checkbox"
              id="prop-is-current"
              v-model="newProperty.is_current"
              class="w-4 h-4 text-blue-600 rounded border-line focus:ring-blue-500"
            />
            <label for="prop-is-current" class="text-sm text-ink-soft cursor-pointer">
              {{ t('settings.properties.setAsPrincipal') }}
            </label>
          </div>
        </div>
        <div v-if="propertyError" class="text-sm text-red-600 dark:text-red-400">{{ propertyError }}</div>
        <Button @click="addProperty" :disabled="!newProperty.name.trim() || propertyLoading">
          {{ propertyLoading ? t('settings.properties.creatingButton') : t('settings.properties.createButton') }}
        </Button>
      </div>
    </Card>

  </div>
</template>

<script setup>
defineOptions({ name: 'PropertiesTab' })

import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import apiClient from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'

const { t } = useI18n()
const allProperties = ref([])
const showAddPropertyForm = ref(false)
const propertyLoading = ref(false)
const propertyError = ref(null)
const newProperty = ref({ name: '', address: '', type: 'owned', is_current: false })

async function fetchAllProperties() {
  try {
    const { data } = await apiClient.get('/properties')
    allProperties.value = data || []
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

async function addProperty() {
  if (!newProperty.value.name.trim()) return
  propertyLoading.value = true
  propertyError.value = null
  try {
    await apiClient.post('/properties', {
      name: newProperty.value.name.trim(),
      address: newProperty.value.address.trim(),
      type: newProperty.value.type,
      is_current: newProperty.value.is_current
    })
    newProperty.value = { name: '', address: '', type: 'owned', is_current: false }
    showAddPropertyForm.value = false
    await fetchAllProperties()
  } catch (err) {
    propertyError.value = err.response?.data?.error || t('settings.properties.createError')
  } finally {
    propertyLoading.value = false
  }
}

onMounted(() => {
  fetchAllProperties()
})
</script>
