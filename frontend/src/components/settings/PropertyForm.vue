<template>
  <form
    class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-xl border border-blue-200 dark:border-blue-800 space-y-3"
    @submit.prevent="emit('submit')"
  >
    <h3 class="text-sm font-medium text-ink">{{ title }}</h3>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <input
        v-model="model.name"
        type="text"
        :placeholder="t('settings.properties.namePlaceholder')"
        :aria-label="t('settings.properties.namePlaceholder')"
        :class="fieldClass"
      />
      <input
        v-model="model.address"
        type="text"
        :placeholder="t('settings.properties.addressPlaceholder')"
        :aria-label="t('settings.properties.addressPlaceholder')"
        autocomplete="street-address"
        :class="fieldClass"
      />
      <select v-model="model.type" :aria-label="t('settings.properties.typeLabel')" :class="fieldClass">
        <option value="owned">{{ t('settings.properties.typeOwned') }}</option>
        <option value="rented">{{ t('settings.properties.typeRented') }}</option>
      </select>
      <label class="flex items-center gap-3 text-sm text-ink-soft">
        <span class="shrink-0">{{ t('settings.properties.residentsLabel') }}</span>
        <input
          v-model.number="model.residents"
          type="number"
          min="1"
          step="1"
          inputmode="numeric"
          :class="[fieldClass, 'w-24']"
        />
      </label>
      <div class="flex items-center gap-2 sm:col-span-2">
        <input
          :id="`${idPrefix}-is-current`"
          v-model="model.is_current"
          type="checkbox"
          :disabled="principalLocked"
          class="w-4 h-4 text-blue-600 rounded border-line focus:ring-blue-500 disabled:opacity-60"
        />
        <label :for="`${idPrefix}-is-current`" class="text-sm text-ink-soft cursor-pointer">
          {{ t('settings.properties.setAsPrincipal') }}
        </label>
      </div>
    </div>
    <div v-if="error" class="text-sm text-danger-soft">{{ error }}</div>
    <div class="flex flex-wrap gap-2">
      <Button type="submit" :disabled="!isValid || loading">
        {{ loading ? loadingLabel : submitLabel }}
      </Button>
      <Button v-if="cancellable" type="button" variant="secondary" @click="emit('cancel')">
        {{ t('common.actions.cancel') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Button from '@/components/common/Button.vue'

defineProps({
  title: { type: String, required: true },
  idPrefix: { type: String, required: true },
  submitLabel: { type: String, required: true },
  loadingLabel: { type: String, required: true },
  loading: { type: Boolean, default: false },
  error: { type: String, default: null },
  cancellable: { type: Boolean, default: false },
  // A home that is already the primary one can't be un-ticked: another home
  // becomes primary by ticking it there.
  principalLocked: { type: Boolean, default: false }
})

const emit = defineEmits(['submit', 'cancel'])
const model = defineModel({ type: Object, required: true })
const { t } = useI18n()

const fieldClass = `px-3 py-3 border border-line rounded-lg bg-surface text-ink text-base
  focus:outline-none focus:ring-2 focus:ring-blue-500`

const isValid = computed(() =>
  model.value.name.trim() !== '' && Number.isInteger(model.value.residents) && model.value.residents >= 1
)
</script>
