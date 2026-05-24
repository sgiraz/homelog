<template>
  <Card class="p-6 border-red-200 dark:border-red-800">
    <h2 class="text-xl font-bold text-red-600 dark:text-red-400 mb-2">{{ t('settings.deleteAccount.title') }}</h2>
    <p class="text-sm text-ink-soft mb-4">
      {{ t('settings.deleteAccount.description') }}
    </p>

    <!-- Step 0: Initial button -->
    <div v-if="deleteStep === 0">
      <Button variant="danger" @click="startDeleteAccount">
        {{ t('settings.deleteAccount.startButton') }}
      </Button>
    </div>

    <!-- Loading state -->
    <div v-else-if="deleteStep === 'loading'" class="text-sm text-ink-muted py-4 text-center">
      {{ t('settings.deleteAccount.loading') }}
    </div>

    <!-- Step 1: Blocking — must nominate admins -->
    <div v-else-if="deleteStep === 'blocking'" class="space-y-4">
      <div class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-lg">
        <p class="text-sm font-medium text-amber-800 dark:text-amber-300 mb-2">
          {{ t('settings.deleteAccount.blockingHeader') }}
        </p>
      </div>

      <div v-for="bp in deleteCheckResult.blocking_properties" :key="bp.property_id"
           class="p-3 border border-line rounded-lg space-y-2">
        <div class="font-medium text-ink">{{ bp.property_name }}</div>
        <div class="flex items-center gap-2">
          <select
            v-model="adminNominations[bp.property_id]"
            class="flex-1 px-3 py-2 border border-line rounded-lg
                   bg-surface text-ink text-sm
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option :value="null" disabled>{{ t('settings.deleteAccount.selectMember') }}</option>
            <option v-for="m in bp.members" :key="m.member_id" :value="m.member_id">
              {{ m.name }}
            </option>
          </select>
          <Button
            size="sm"
            :disabled="!adminNominations[bp.property_id] || promotingProperty === bp.property_id"
            @click="handlePromoteAdmin(bp.property_id)"
          >
            {{ promotingProperty === bp.property_id ? t('settings.deleteAccount.nominateSaving') : t('settings.deleteAccount.nominateButton') }}
          </Button>
        </div>
      </div>

      <div class="flex gap-2">
        <Button variant="secondary" size="sm" @click="deleteStep = 0">{{ t('settings.deleteAccount.cancelButton') }}</Button>
      </div>
    </div>

    <!-- Step 2: Can delete — confirm -->
    <div v-else-if="deleteStep === 'confirm'" class="space-y-4">
      <div v-if="deleteCheckResult.data_loss_properties?.length > 0"
           class="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-700 rounded-lg">
        <p class="text-sm font-medium text-red-800 dark:text-red-300 mb-1">
          {{ t('settings.deleteAccount.confirmHeader') }}
        </p>
        <ul class="text-sm text-red-700 dark:text-red-400 list-disc list-inside">
          <li v-for="name in deleteCheckResult.data_loss_properties" :key="name">{{ name }}</li>
        </ul>
      </div>

      <Input
        v-model="deletePassword"
        :label="t('settings.deleteAccount.passwordLabel')"
        type="password"
        :placeholder="t('settings.deleteAccount.passwordPlaceholder')"
        autocomplete="current-password"
      />

      <label class="flex items-start gap-2 cursor-pointer">
        <input v-model="deleteConfirmed" type="checkbox" class="mt-1 rounded border-line" />
        <span class="text-sm text-ink-soft">
          {{ t('settings.deleteAccount.confirmCheckbox') }}
        </span>
      </label>

      <div v-if="deleteError" class="text-red-600 text-sm bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
        {{ deleteError }}
      </div>

      <div class="flex gap-2">
        <Button
          variant="danger"
          :disabled="!deletePassword || !deleteConfirmed || deleteLoading"
          @click="handleDeleteAccount"
        >
          {{ deleteLoading ? t('settings.deleteAccount.deletingButton') : t('settings.deleteAccount.confirmButton') }}
        </Button>
        <Button variant="secondary" @click="deleteStep = 0">{{ t('settings.deleteAccount.cancelButton') }}</Button>
      </div>
    </div>
  </Card>
</template>

<script setup>
defineOptions({ name: 'DeleteAccountSection' })

import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { accountAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import Input from '@/components/common/Input.vue'

const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()

const deleteStep = ref(0) // 0, 'loading', 'blocking', 'confirm'
const deleteCheckResult = ref(null)
const adminNominations = ref({})
const promotingProperty = ref(null)
const deletePassword = ref('')
const deleteConfirmed = ref(false)
const deleteError = ref(null)
const deleteLoading = ref(false)

async function startDeleteAccount() {
  deleteStep.value = 'loading'
  deleteError.value = null
  try {
    const { data } = await accountAPI.deleteCheck()
    deleteCheckResult.value = data
    if (!data.can_delete) {
      adminNominations.value = {}
      for (const bp of data.blocking_properties) {
        adminNominations.value[bp.property_id] = null
      }
      deleteStep.value = 'blocking'
    } else {
      deleteStep.value = 'confirm'
    }
  } catch {
    deleteStep.value = 0
    window.$toast?.error(t('settings.deleteAccount.checkError'))
  }
}

async function handlePromoteAdmin(propertyId) {
  const memberId = adminNominations.value[propertyId]
  if (!memberId) return
  promotingProperty.value = propertyId
  try {
    await accountAPI.promoteAdmin(propertyId, memberId)
    window.$toast?.success(t('settings.deleteAccount.promoteSuccess'))
    await startDeleteAccount()
  } catch (err) {
    window.$toast?.error(err.response?.data?.error || t('settings.deleteAccount.promoteError'))
  } finally {
    promotingProperty.value = null
  }
}

async function handleDeleteAccount() {
  deleteError.value = null
  deleteLoading.value = true
  try {
    await accountAPI.deleteAccount(deletePassword.value)
    authStore.logout()
    router.push('/login')
    window.$toast?.success(t('settings.deleteAccount.accountDeleted'))
  } catch (err) {
    deleteError.value = err.response?.data?.error || t('settings.deleteAccount.deleteGenericError')
  } finally {
    deleteLoading.value = false
  }
}
</script>
