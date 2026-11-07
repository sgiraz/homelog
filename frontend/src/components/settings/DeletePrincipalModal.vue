<template>
  <BaseModal :title="t('settings.properties.deleteConfirmTitle', { name: property.name })" @close="emit('close')">
    <form class="space-y-4" @submit.prevent="submit">
      <p class="text-ink-soft">{{ t('settings.properties.deleteConfirmMessage') }}</p>

      <fieldset class="space-y-2">
        <legend class="text-sm font-medium text-ink mb-2">{{ t('settings.properties.newPrincipalLabel') }}</legend>
        <label
          v-for="c in candidates"
          :key="c.id"
          class="flex items-center gap-3 p-3 rounded-xl border cursor-pointer transition-colors"
          :class="selectedId === c.id ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20' : 'border-line bg-surface hover:bg-surface-2'"
        >
          <input
            v-model="selectedId"
            type="radio"
            name="new-principal"
            :value="c.id"
            class="w-4 h-4 text-blue-600 border-line focus:ring-blue-500"
          />
          <span class="flex-1 min-w-0">
            <span class="block font-medium text-ink truncate">{{ c.name }}</span>
            <span v-if="c.address" class="block text-xs text-ink-muted truncate">{{ c.address }}</span>
          </span>
        </label>
      </fieldset>

      <div class="flex gap-3 justify-end">
        <Button variant="secondary" @click="emit('close')">{{ t('common.actions.cancel') }}</Button>
        <Button type="submit" variant="danger" :disabled="!selectedId || loading">
          {{ t('common.actions.delete') }}
        </Button>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseModal from '@/components/common/BaseModal.vue'
import Button from '@/components/common/Button.vue'

const props = defineProps({
  property: { type: Object, required: true },
  // The owner's other homes, one of which becomes the primary one.
  candidates: { type: Array, required: true },
  loading: { type: Boolean, default: false }
})

const emit = defineEmits(['close', 'confirm'])
const { t } = useI18n()

// No default pick unless the choice is forced: the owner decides.
const selectedId = ref(props.candidates.length === 1 ? props.candidates[0].id : null)

function submit() {
  if (selectedId.value) emit('confirm', selectedId.value)
}
</script>
