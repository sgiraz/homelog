<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-start justify-center z-[60] p-4 pt-8 overflow-y-auto"
    @click.self="$emit('close')"
  >
    <Card class="w-full max-w-lg p-6 my-auto">
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-xl font-bold text-ink">{{ t('utilities.templatesManager.title') }}</h3>
        <button @click="$emit('close')" class="text-ink-muted hover:text-ink">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <p class="text-ink-soft text-sm mb-4">
        {{ t('utilities.templatesManager.intro') }}
      </p>

      <!-- Loading -->
      <div v-if="loading" class="text-center py-8 text-ink-soft">
        {{ t('utilities.templatesManager.loading') }}
      </div>

      <!-- Empty State -->
      <div v-else-if="templates.length === 0" class="text-center py-8">
        <svg class="w-12 h-12 mx-auto text-ink-faint" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
        <p class="mt-3 text-ink-soft">{{ t('utilities.templatesManager.empty') }}</p>
        <p class="mt-1 text-sm text-ink-muted">
          {{ t('utilities.templatesManager.emptyHint') }}
        </p>
      </div>

      <!-- Templates List -->
      <div v-else class="space-y-3 mb-4">
        <div
          v-for="tpl in templates"
          :key="tpl.id"
          class="flex items-center justify-between p-4 border border-line rounded-lg hover:bg-surface-2 transition-colors"
        >
          <div class="flex items-center gap-3">
            <div :class="[
              'p-2 rounded-lg',
              utilityTypeStyle(tpl.utility_type).tile
            ]">
              <span class="text-xl">{{ utilityTypeStyle(tpl.utility_type).icon }}</span>
            </div>
            <div>
              <div class="flex items-center gap-2">
                <span class="font-medium text-ink">{{ tpl.name }}</span>
                <Badge v-if="tpl.is_default" variant="info">
                  {{ t('utilities.templatesManager.defaultBadge') }}
                </Badge>
              </div>
              <p class="text-sm text-ink-muted">{{ tpl.provider }}</p>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <button
              @click="exportTemplate(tpl)"
              class="p-2 text-ink-faint hover:text-blue-500 transition-colors"
              :title="t('utilities.templatesManager.exportTitle')"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
            </button>
            <button
              @click="editTemplate(tpl)"
              class="p-2 text-ink-faint hover:text-blue-500 transition-colors"
              :title="t('utilities.templatesManager.editTitle')"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </button>
            <button
              @click="confirmDelete(tpl)"
              class="p-2 text-ink-faint hover:text-danger-soft transition-colors"
              :title="t('utilities.templatesManager.deleteTitle')"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      </div>

      <Button @click="showWizard = true" class="w-full">
        <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        {{ t('utilities.templatesManager.createButton') }}
      </Button>
      <Button variant="secondary" @click="importInput.click()" :disabled="importing" class="w-full mt-2">
        {{ importing ? t('utilities.templatesManager.importing') : t('utilities.templatesManager.importButton') }}
      </Button>
      <input ref="importInput" type="file" accept=".json,application/json" class="hidden" @change="importTemplate" />
      <p class="mt-2 text-xs text-ink-muted text-center">{{ t('utilities.templatesManager.importHint') }}</p>

      <!-- Delete Confirmation -->
      <div
        v-if="templateToDelete"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-[70]"
        @click.self="templateToDelete = null"
      >
        <Card class="w-full max-w-sm p-6 mx-4">
          <h4 class="text-lg font-bold text-ink mb-2">{{ t('utilities.templatesManager.deleteConfirmTitle') }}</h4>
          <p class="text-ink-soft mb-4">
            {{ t('utilities.templatesManager.deleteConfirmMessage', { name: templateToDelete.name }) }}
          </p>
          <div class="flex gap-3">
            <Button variant="secondary" @click="templateToDelete = null" class="flex-1">
              {{ t('utilities.templatesManager.cancel') }}
            </Button>
            <Button variant="danger" @click="deleteTemplate" :disabled="deleting" class="flex-1">
              {{ deleting ? t('utilities.templatesManager.deleting') : t('utilities.templatesManager.delete') }}
            </Button>
          </div>
        </Card>
      </div>
    </Card>

    <!-- Template Wizard -->
    <TemplateWizard
      v-if="showWizard"
      :existing-template="editingTemplate"
      :property-id="propertyId"
      @close="closeWizard"
      @saved="onTemplateSaved"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { utilityTypeStyle } from '@/config/utilityTypes'
import { useI18n } from 'vue-i18n'
import Badge from '@/components/common/Badge.vue'
import { templatesAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import TemplateWizard from './TemplateWizard.vue'
import { apiErrorMessage } from '@/utils/apiError'

// Must match billTemplateFileFormat in handlers/pdf.go.
const TEMPLATE_FILE_FORMAT = 'homelog-bill-template/1'

const props = defineProps({
  propertyId: { type: Number, default: null }
})

defineEmits(['close'])

const { t } = useI18n()

const loading = ref(true)
const templates = ref([])
const showWizard = ref(false)
const editingTemplate = ref(null)
const templateToDelete = ref(null)
const deleting = ref(false)
const importInput = ref(null)
const importing = ref(false)

async function fetchTemplates() {
  loading.value = true
  try {
    const { data } = await templatesAPI.listBillTemplates({ property_id: props.propertyId })
    templates.value = data || []
  } catch (err) {
    console.error('Error fetching templates:', err)
    templates.value = []
  } finally {
    loading.value = false
  }
}

function editTemplate(tpl) {
  editingTemplate.value = tpl
  showWizard.value = true
}

function confirmDelete(tpl) {
  templateToDelete.value = tpl
}

async function deleteTemplate() {
  if (!templateToDelete.value) return

  deleting.value = true
  try {
    await templatesAPI.deleteBillTemplate(templateToDelete.value.id)
    await fetchTemplates()
    templateToDelete.value = null
  } catch (err) {
    console.error('Error deleting template:', err)
  } finally {
    deleting.value = false
  }
}

function exportTemplate(tpl) {
  const file = {
    format: TEMPLATE_FILE_FORMAT,
    name: tpl.name,
    provider: tpl.provider,
    utility_type: tpl.utility_type,
    extraction_rules: JSON.parse(tpl.extraction_rules || '[]')
  }
  const url = URL.createObjectURL(new Blob([JSON.stringify(file, null, 2)], { type: 'application/json' }))
  const a = document.createElement('a')
  a.href = url
  a.download = `${tpl.provider}-${tpl.utility_type}.json`.replace(/[^\w.-]+/g, '_')
  a.click()
  URL.revokeObjectURL(url)
}

async function importTemplate(e) {
  const file = e.target.files[0]
  e.target.value = ''
  if (!file) return

  let parsed
  try {
    parsed = JSON.parse(await file.text())
  } catch {
    window.$toast?.error(t('errors.template_file_invalid'))
    return
  }

  importing.value = true
  try {
    await templatesAPI.importBillTemplate({ ...parsed, property_id: props.propertyId })
    window.$toast?.success(t('utilities.templatesManager.importSuccess', { name: parsed.name }))
    await fetchTemplates()
  } catch (err) {
    window.$toast?.error(apiErrorMessage(err, t('utilities.templatesManager.importError')))
  } finally {
    importing.value = false
  }
}

function closeWizard() {
  showWizard.value = false
  editingTemplate.value = null
}

function onTemplateSaved() {
  closeWizard()
  fetchTemplates()
}

onMounted(() => {
  fetchTemplates()
})
</script>
