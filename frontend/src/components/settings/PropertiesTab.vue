<template>
  <div class="space-y-4">
    <Card class="p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h2 class="text-xl font-bold text-ink">{{ t('settings.properties.title') }}</h2>
          <p class="text-sm text-ink-soft mt-1">{{ t('settings.properties.subtitle') }}</p>
        </div>
        <Button
          @click="toggleAddForm"
          :variant="showAddPropertyForm ? 'secondary' : 'primary'"
          size="sm"
          :aria-label="showAddPropertyForm ? t('common.actions.cancel') : t('settings.properties.newTitle')"
        >
          <svg v-if="!showAddPropertyForm" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </Button>
      </div>

      <div v-if="allProperties.length > 0" class="space-y-2 mb-4">
        <template v-for="prop in allProperties" :key="prop.id">
          <PropertyForm
            v-if="editingId === prop.id"
            v-model="editForm"
            :title="t('settings.properties.editTitle')"
            :id-prefix="`prop-edit-${prop.id}`"
            :submit-label="t('common.actions.save')"
            :loading-label="t('settings.properties.savingButton')"
            :loading="editLoading"
            :error="editError"
            :principal-locked="prop.is_current || !!prop.end_date"
            cancellable
            @submit="saveEdit(prop)"
            @cancel="cancelEdit"
          />
          <div
            v-else
            class="flex items-center gap-3 p-3 rounded-xl border border-line bg-surface"
          >
            <span class="text-2xl">🏠</span>
            <div class="flex-1 min-w-0">
              <div class="font-medium text-ink truncate">{{ prop.name }}</div>
              <!-- Badge on the second line: beside the name it starves it on a phone. -->
              <div v-if="prop.is_current || prop.end_date || prop.address || manyHouseholds" class="flex items-center gap-2 min-w-0 mt-0.5">
                <Badge v-if="prop.is_current" variant="positive" class="shrink-0">{{ t('settings.properties.principalBadge') }}</Badge>
                <Badge v-if="prop.end_date" variant="neutral" class="shrink-0">{{ t('settings.properties.archivedBadge') }}</Badge>
                <span v-if="manyHouseholds" class="text-xs text-ink-muted truncate shrink-0">{{ prop.household?.name }}</span>
                <span v-if="prop.address" class="text-xs text-ink-muted truncate">{{ prop.address }}</span>
              </div>
            </div>
            <div v-if="isOwner(prop)" class="flex items-center">
              <!-- Archive: the family moved out, the history stays. The primary
                   home is always one in use, so it has no archive action. -->
              <button
                v-if="!prop.is_current"
                type="button"
                class="p-2 rounded-lg text-ink-muted hover:text-ink hover:bg-surface-2 transition-colors disabled:opacity-50"
                :aria-label="t(prop.end_date ? 'settings.properties.restoreAria' : 'settings.properties.archiveAria', { name: prop.name })"
                :title="t(prop.end_date ? 'settings.properties.restore' : 'settings.properties.archive')"
                :disabled="archivingId === prop.id"
                @click="toggleArchived(prop)"
              >
                <svg v-if="!prop.end_date" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
                </svg>
                <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
              </button>
              <button
                type="button"
                class="p-2 rounded-lg text-ink-muted hover:text-ink hover:bg-surface-2 transition-colors"
                :aria-label="t('settings.properties.editAria', { name: prop.name })"
                :title="t('common.actions.edit')"
                @click="startEdit(prop)"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536M4 20h4l10.5-10.5a2.5 2.5 0 00-3.536-3.536L4 16.464V20z" />
                </svg>
              </button>
              <button
                v-if="allProperties.length > 1"
                type="button"
                class="p-2 rounded-lg text-ink-muted hover:text-danger-soft hover:bg-surface-2 transition-colors disabled:opacity-50"
                :aria-label="t('settings.properties.deleteAria', { name: prop.name })"
                :title="t('common.actions.delete')"
                :disabled="deletingId === prop.id"
                @click="deleteProperty(prop)"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>
        </template>
      </div>

      <PropertyForm
        v-if="showAddPropertyForm"
        v-model="newProperty"
        :title="t('settings.properties.newTitle')"
        id-prefix="prop-new"
        :submit-label="t('settings.properties.createButton')"
        :loading-label="t('settings.properties.creatingButton')"
        :loading="propertyLoading"
        :error="propertyError"
        creating
        :households="adminHouseholds"
        @submit="addProperty"
      />
    </Card>

    <DeletePrincipalModal
      v-if="principalToDelete"
      :property="principalToDelete"
      :candidates="successorCandidates"
      :loading="deletingId === principalToDelete.id"
      @close="principalToDelete = null"
      @confirm="(newPrincipalId) => performDelete(principalToDelete, newPrincipalId)"
    />
  </div>
</template>

<script setup>
defineOptions({ name: 'PropertiesTab' })

import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Badge from '@/components/common/Badge.vue'
import { propertiesAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import PropertyForm from '@/components/settings/PropertyForm.vue'
import DeletePrincipalModal from '@/components/settings/DeletePrincipalModal.vue'
import { apiErrorMessage } from '@/utils/apiError'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { useViewCache } from '@/composables/useViewCache'

const { t } = useI18n()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()
const { evictCachedViews } = useViewCache()

const blankProperty = () => ({ name: '', address: '', type: 'owned', residents: 1, is_current: false, household_id: null })

// The households a new home can join: the ones the user administers, the
// active one first, since moving home within the same family is the usual case.
const adminHouseholds = computed(() => {
  const admin = settingsStore.households.filter((h) => h.is_admin)
  return [...admin].sort((a, b) => (b.id === settingsStore.householdId) - (a.id === settingsStore.householdId))
})
const manyHouseholds = computed(() => settingsStore.households.length > 1)

function newPropertyDraft() {
  return { ...blankProperty(), household_id: adminHouseholds.value[0]?.id ?? null }
}

const allProperties = ref([])
const showAddPropertyForm = ref(false)
const propertyLoading = ref(false)
const propertyError = ref(null)
const newProperty = ref(blankProperty())

const editingId = ref(null)
const editForm = ref(blankProperty())
const editLoading = ref(false)
const editError = ref(null)
const deletingId = ref(null)
const principalToDelete = ref(null)

// The primary home's successor comes from the same household, among the
// homes still in use.
function householdSiblings(prop) {
  return allProperties.value.filter((p) => p.household_id === prop.household_id && p.id !== prop.id)
}
const successorCandidates = computed(() =>
  principalToDelete.value ? householdSiblings(principalToDelete.value).filter((p) => !p.end_date) : []
)

const archivingId = ref(null)

async function toggleArchived(prop) {
  const archive = !prop.end_date
  archivingId.value = prop.id
  try {
    await propertiesAPI.update(prop.id, { archived: archive })
    window.$toast?.success(t(archive ? 'settings.properties.archiveSuccess' : 'settings.properties.restoreSuccess', { name: prop.name }))
    await fetchAllProperties()
    // Archived homes leave the expense form: views must load the list again.
    evictCachedViews(['SettingsView'])
  } catch (err) {
    window.$toast?.error(apiErrorMessage(err, t('settings.properties.updateError')))
  } finally {
    archivingId.value = null
  }
}

// Only the creator may edit or delete a home (the API filters on user_id);
// members see it read-only.
function isOwner(prop) {
  return prop.user_id === authStore.user?.id
}

async function fetchAllProperties() {
  try {
    const { data } = await propertiesAPI.list()
    allProperties.value = data || []
  } catch (err) {
    console.error('Error fetching properties:', err)
  }
}

// A primary home or a household changed or vanished: re-resolve them in the
// store and drop cached views that captured the old ones at mount.
async function refreshCurrentProperty() {
  await settingsStore.loadHouseholdSettings()
  evictCachedViews(['SettingsView'])
}

function toggleAddForm() {
  showAddPropertyForm.value = !showAddPropertyForm.value
  if (showAddPropertyForm.value) {
    cancelEdit()
    newProperty.value = newPropertyDraft()
  }
}

async function addProperty() {
  if (!newProperty.value.name.trim()) return
  propertyLoading.value = true
  propertyError.value = null
  try {
    await propertiesAPI.create({
      name: newProperty.value.name.trim(),
      address: newProperty.value.address.trim(),
      type: newProperty.value.type,
      residents: newProperty.value.residents,
      is_current: newProperty.value.is_current,
      household_id: newProperty.value.household_id ?? undefined
    })
    newProperty.value = blankProperty()
    showAddPropertyForm.value = false
    await fetchAllProperties()
    // A new household, or a new primary home: the store has to know.
    await refreshCurrentProperty()
  } catch (err) {
    propertyError.value = apiErrorMessage(err, t('settings.properties.createError'))
  } finally {
    propertyLoading.value = false
  }
}

function startEdit(prop) {
  showAddPropertyForm.value = false
  editError.value = null
  editingId.value = prop.id
  editForm.value = {
    name: prop.name,
    address: prop.address || '',
    type: prop.type === 'rented' ? 'rented' : 'owned',
    residents: prop.residents || 1,
    is_current: prop.is_current
  }
}

function cancelEdit() {
  editingId.value = null
  editError.value = null
}

async function saveEdit(prop) {
  editLoading.value = true
  editError.value = null
  const becomesPrincipal = editForm.value.is_current && !prop.is_current
  try {
    await propertiesAPI.update(prop.id, {
      name: editForm.value.name.trim(),
      address: editForm.value.address.trim(),
      type: editForm.value.type,
      residents: editForm.value.residents,
      is_current: editForm.value.is_current
    })
    editingId.value = null
    await fetchAllProperties()
    if (becomesPrincipal) await refreshCurrentProperty()
  } catch (err) {
    editError.value = apiErrorMessage(err, t('settings.properties.updateError'))
  } finally {
    editLoading.value = false
  }
}

// A shared home cannot be deleted (the API answers property_has_members): say
// so before asking for a confirmation that would lead nowhere.
// Deleting the primary home asks which of the household's other homes takes
// its place; the only home of a household takes the household with it.
async function deleteProperty(prop) {
  if (prop.is_shared) {
    window.$toast?.info(t('errors.property_has_members'))
    return
  }
  const siblings = householdSiblings(prop)
  if (prop.is_current && siblings.length > 0) {
    // Its place goes to a home in use: with only archived ones left, one has
    // to be restored first.
    if (!siblings.some((p) => !p.end_date)) {
      window.$toast?.info(t('settings.properties.deleteNeedsActiveHome'))
      return
    }
    principalToDelete.value = prop
    return
  }
  const ok = await confirm({
    title: t('settings.properties.deleteConfirmTitle', { name: prop.name }),
    message: siblings.length > 0
      ? t('settings.properties.deleteKeepsHouseholdMessage')
      : t('settings.properties.deleteConfirmMessage'),
    confirmText: t('common.actions.delete'),
    variant: 'danger'
  })
  if (ok) await performDelete(prop)
}

async function performDelete(prop, newPrincipalId = null) {
  deletingId.value = prop.id
  try {
    await propertiesAPI.delete(prop.id, newPrincipalId ? { new_principal_id: newPrincipalId } : undefined)
    principalToDelete.value = null
    if (editingId.value === prop.id) cancelEdit()
    window.$toast?.success(t('settings.properties.deleteSuccess', { name: prop.name }))
    await fetchAllProperties()
    await refreshCurrentProperty()
  } catch (err) {
    window.$toast?.error(apiErrorMessage(err, t('settings.properties.deleteError')))
  } finally {
    deletingId.value = null
  }
}

onMounted(() => {
  fetchAllProperties()
})
</script>
