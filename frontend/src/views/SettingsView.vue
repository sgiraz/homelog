<template>
  <div class="space-y-4">
    <!-- Profile Card (Apple HIG: always visible above tabs) -->
    <Card class="p-4 sm:p-6">
      <div class="flex items-center gap-4">
        <!-- Avatar with always-visible camera badge -->
        <div class="relative shrink-0">
          <button
            @click="$refs.avatarInput.click()"
            class="block relative cursor-pointer"
            :aria-label="t('settings.profile.changeAvatar')"
          >
            <img
              v-if="authStore.avatarUrl"
              :src="authStore.avatarUrl"
              :alt="authStore.user?.name"
              class="w-16 h-16 sm:w-20 sm:h-20 rounded-full object-cover"
            />
            <div
              v-else
              class="w-16 h-16 sm:w-20 sm:h-20 rounded-full bg-gradient-to-br from-blue-500 to-purple-600
                      flex items-center justify-center text-white text-xl sm:text-2xl font-bold"
            >
              {{ userInitials }}
            </div>
            <!-- Camera badge (always visible, Apple-style) -->
            <div
              class="absolute -bottom-0.5 -right-0.5 w-7 h-7 sm:w-8 sm:h-8
                     bg-gray-100 dark:bg-gray-600 rounded-full
                     flex items-center justify-center
                     border-2 border-white dark:border-gray-800
                     shadow-sm"
              :class="avatarUploading ? 'animate-pulse' : ''"
            >
              <svg v-if="!avatarUploading" class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-gray-600 dark:text-gray-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              <div v-else class="w-3.5 h-3.5 border-2 border-gray-600 dark:border-gray-200 border-t-transparent rounded-full animate-spin" />
            </div>
          </button>
          <input
            ref="avatarInput"
            type="file"
            accept="image/jpeg,image/png,image/webp"
            class="hidden"
            @change="onAvatarSelected"
          />
        </div>
        <div class="flex-1 min-w-0">
          <div class="font-semibold text-lg text-gray-900 dark:text-white truncate">{{ authStore.user?.name }}</div>
          <div class="text-sm text-gray-500 dark:text-gray-400 truncate">{{ authStore.user?.email }}</div>
          <div class="flex items-center gap-3 mt-1.5">
            <span class="text-xs text-gray-400 dark:text-gray-500">
              {{ settingsStore.isPropertyAdmin ? t('settings.profile.roleAdmin') : t('settings.profile.roleUser') }}
            </span>
            <button
              v-if="authStore.avatarUrl"
              @click="removeAvatar"
              class="text-xs text-red-500 hover:text-red-700 dark:hover:text-red-400"
            >
              {{ t('settings.profile.removeAvatar') }}
            </button>
          </div>
        </div>
      </div>
    </Card>

    <!-- Tab Bar -->
    <div class="overflow-x-auto -mx-4 sm:mx-0 px-4 sm:px-0 pb-1">
      <div class="flex gap-1 min-w-max sm:min-w-0 sm:flex-wrap">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          @click="activeTab = tab.id"
          :class="[
            'flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium whitespace-nowrap transition-colors',
            activeTab === tab.id
              ? 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
          ]"
        >
          <span>{{ tab.icon }}</span>
          <span>{{ tab.label }}</span>
        </button>
      </div>
    </div>

    <!-- Tab: Famiglia -->
    <FamilyTab v-show="activeTab === 'famiglia'" />

    <!-- Tab: Proprietà -->
    <PropertiesTab v-show="activeTab === 'proprieta'" />

    <!-- Tab: Preferenze -->
    <PreferencesTab v-show="activeTab === 'preferenze'" />

    <!-- Tab: Categorie -->
    <CategoriesTab v-show="activeTab === 'categorie'" />

    <!-- Tab: Dati -->
    <DataTab v-show="activeTab === 'dati'" />

    <!-- Avatar Crop Modal -->
    <AvatarCropModal
      v-if="showCropModal"
      :image-src="cropImageSrc"
      @close="showCropModal = false; cropImageSrc = null"
      @cropped="onAvatarCropped"
    />
  </div>
</template>

<script setup>
defineOptions({ name: 'SettingsView' })

import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { avatarAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import AvatarCropModal from '@/components/common/AvatarCropModal.vue'
import FamilyTab from '@/components/settings/FamilyTab.vue'
import PropertiesTab from '@/components/settings/PropertiesTab.vue'
import PreferencesTab from '@/components/settings/PreferencesTab.vue'
import CategoriesTab from '@/components/settings/CategoriesTab.vue'
import DataTab from '@/components/settings/DataTab.vue'

const { t } = useI18n()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

// Tabs — IDs are stable (used as route query) so labels are translated separately.
const activeTab = ref('famiglia')
const tabs = computed(() => [
  { id: 'famiglia',   label: t('settings.tabs.family'),      icon: '👥' },
  { id: 'proprieta',  label: t('settings.tabs.properties'),  icon: '🏠' },
  { id: 'preferenze', label: t('settings.tabs.preferences'), icon: '⚙️' },
  { id: 'categorie',  label: t('settings.tabs.categories'),  icon: '🏷️' },
  { id: 'dati',       label: t('settings.tabs.data'),        icon: '📦' },
])

const userInitials = computed(() => {
  const name = authStore.user?.name || 'U'
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
})

// ── Avatar ──────────────────────────────────────────────────────────────────

const avatarUploading = ref(false)
const avatarInput = ref(null)
const showCropModal = ref(false)
const cropImageSrc = ref(null)

function onAvatarSelected(e) {
  const file = e.target.files[0]
  if (!file) return
  if (file.size > 5 * 1024 * 1024) {
    window.$toast?.error(t('settings.profile.avatarTooLarge'))
    if (avatarInput.value) avatarInput.value.value = ''
    return
  }
  // Read file and open crop modal
  const reader = new FileReader()
  reader.onload = (ev) => {
    cropImageSrc.value = ev.target.result
    showCropModal.value = true
  }
  reader.readAsDataURL(file)
  if (avatarInput.value) avatarInput.value.value = ''
}

async function onAvatarCropped(file) {
  showCropModal.value = false
  cropImageSrc.value = null
  avatarUploading.value = true
  try {
    const { data } = await avatarAPI.upload(file)
    authStore.updateUser(data.user)
    window.$toast?.success(t('settings.profile.avatarUploaded'))
  } catch (err) {
    window.$toast?.error(err.response?.data?.error || t('settings.profile.avatarUploadError'))
  } finally {
    avatarUploading.value = false
  }
}

async function removeAvatar() {
  const ok = await confirm({
    title: t('settings.profile.confirmRemoveTitle'),
    message: t('settings.profile.confirmRemoveMessage'),
    confirmText: t('settings.profile.confirmRemoveAction'),
    variant: 'danger'
  })
  if (!ok) return
  try {
    const { data } = await avatarAPI.delete()
    authStore.updateUser(data.user)
    window.$toast?.success(t('settings.profile.avatarRemoved'))
  } catch {
    window.$toast?.error(t('settings.profile.avatarRemoveError'))
  }
}
</script>
