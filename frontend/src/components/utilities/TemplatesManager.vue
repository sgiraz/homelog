<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-start justify-center z-[60] p-4 pt-8 overflow-y-auto"
    @click.self="$emit('close')"
  >
    <Card class="w-full max-w-lg p-6 my-auto">
      <div class="flex items-center justify-between mb-6">
        <h3 class="text-xl font-bold text-gray-900 dark:text-white">Template Estrazione</h3>
        <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <p class="text-gray-600 dark:text-gray-400 text-sm mb-4">
        I template permettono di estrarre automaticamente i dati dalle bollette PDF di diversi fornitori.
      </p>

      <!-- Loading -->
      <div v-if="loading" class="text-center py-8 text-gray-600 dark:text-gray-400">
        Caricamento...
      </div>

      <!-- Empty State -->
      <div v-else-if="templates.length === 0" class="text-center py-8">
        <svg class="w-12 h-12 mx-auto text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
        <p class="mt-3 text-gray-600 dark:text-gray-400">Nessun template creato</p>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-500">
          Crea un template per estrarre automaticamente i dati dalle bollette
        </p>
      </div>

      <!-- Templates List -->
      <div v-else class="space-y-3 mb-4">
        <div
          v-for="tpl in templates"
          :key="tpl.id"
          class="flex items-center justify-between p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
        >
          <div class="flex items-center gap-3">
            <div :class="[
              'p-2 rounded-lg',
              getUtilityBgClass(tpl.utility_type)
            ]">
              <span class="text-xl">{{ getUtilityIcon(tpl.utility_type) }}</span>
            </div>
            <div>
              <div class="flex items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ tpl.name }}</span>
                <span v-if="tpl.is_default" class="text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 px-2 py-0.5 rounded-full">
                  Default
                </span>
              </div>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ tpl.provider }}</p>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <button
              @click="editTemplate(tpl)"
              class="p-2 text-gray-400 hover:text-blue-500 transition-colors"
              title="Modifica"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </button>
            <button
              @click="confirmDelete(tpl)"
              class="p-2 text-gray-400 hover:text-red-500 transition-colors"
              title="Elimina"
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
        Crea Nuovo Template
      </Button>

      <!-- Delete Confirmation -->
      <div
        v-if="templateToDelete"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-[70]"
        @click.self="templateToDelete = null"
      >
        <Card class="w-full max-w-sm p-6 mx-4">
          <h4 class="text-lg font-bold text-gray-900 dark:text-white mb-2">Conferma Eliminazione</h4>
          <p class="text-gray-600 dark:text-gray-400 mb-4">
            Vuoi eliminare il template "{{ templateToDelete.name }}"? Questa azione non puo essere annullata.
          </p>
          <div class="flex gap-3">
            <Button variant="secondary" @click="templateToDelete = null" class="flex-1">
              Annulla
            </Button>
            <Button variant="danger" @click="deleteTemplate" :disabled="deleting" class="flex-1">
              {{ deleting ? 'Eliminazione...' : 'Elimina' }}
            </Button>
          </div>
        </Card>
      </div>
    </Card>

    <!-- Template Wizard -->
    <TemplateWizard
      v-if="showWizard"
      :existing-template="editingTemplate"
      @close="closeWizard"
      @saved="onTemplateSaved"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { templatesAPI } from '@/api/client'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import TemplateWizard from './TemplateWizard.vue'

const emit = defineEmits(['close'])

const loading = ref(true)
const templates = ref([])
const showWizard = ref(false)
const editingTemplate = ref(null)
const templateToDelete = ref(null)
const deleting = ref(false)

function getUtilityIcon(type) {
  const icons = {
    electricity: '\u26A1', gas: '\uD83D\uDD25', water: '\uD83D\uDCA7', waste: '\u267B\uFE0F',
    internet: '\uD83C\uDF10', insurance: '\uD83D\uDEE1\uFE0F', affitto: '\uD83C\uDFE0', mutuo: '\uD83C\uDFE6'
  }
  return icons[type] || '\u26A1'
}

function getUtilityBgClass(type) {
  const classes = {
    electricity: 'bg-yellow-100 dark:bg-yellow-900/30',
    gas: 'bg-orange-100 dark:bg-orange-900/30',
    water: 'bg-blue-100 dark:bg-blue-900/30',
    waste: 'bg-green-100 dark:bg-green-900/30',
    internet: 'bg-indigo-100 dark:bg-indigo-900/30',
    insurance: 'bg-emerald-100 dark:bg-emerald-900/30',
    affitto: 'bg-purple-100 dark:bg-purple-900/30',
    mutuo: 'bg-sky-100 dark:bg-sky-900/30',
  }
  return classes[type] || classes.electricity
}

async function fetchTemplates() {
  loading.value = true
  try {
    const { data } = await templatesAPI.listBillTemplates()
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
