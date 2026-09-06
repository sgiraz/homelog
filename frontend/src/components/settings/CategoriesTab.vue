<template>
  <div class="space-y-4">
    <Card class="p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h2 class="text-xl font-bold text-ink">{{ t('settings.categories.title') }}</h2>
          <p class="text-sm text-ink-soft mt-1">
            {{ t('settings.categories.subtitle') }}
          </p>
        </div>
        <Button v-if="!showAddCategoryForm" @click="showAddCategoryForm = true" size="sm">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
        </Button>
      </div>

      <!-- Add Category Form -->
      <div v-if="showAddCategoryForm" class="mb-4 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-xl border border-blue-200 dark:border-blue-800 overflow-hidden">
        <h3 class="text-sm font-medium text-ink mb-3">{{ t('settings.categories.newCategoryTitle') }}</h3>
        <div class="flex gap-2 mb-2 min-w-0">
          <input
            v-model="newCategory.icon"
            type="text"
            :placeholder="t('settings.categories.iconPlaceholder')"
            maxlength="4"
            class="w-14 shrink-0 px-2 py-3 border border-line rounded-lg
                   bg-surface text-ink text-center text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <input
            v-model="newCategory.name"
            type="text"
            :placeholder="t('settings.categories.namePlaceholder')"
            class="flex-1 min-w-0 px-3 py-3 border border-line rounded-lg
                   bg-surface text-ink text-base
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
            @keyup.enter="addCategory"
          />
        </div>
        <div v-if="isAdmin" class="flex items-center gap-2 mb-3">
          <input
            type="checkbox"
            id="cat-is-default"
            v-model="newCategory.is_default"
            class="w-4 h-4 text-blue-600 rounded border-line focus:ring-blue-500"
          />
          <label for="cat-is-default" class="text-sm text-ink-soft cursor-pointer">
            {{ t('settings.categories.isDefault') }}
          </label>
        </div>
        <div class="flex gap-2">
          <Button @click="addCategory" :disabled="!newCategory.name.trim()">{{ t('settings.categories.save') }}</Button>
          <Button variant="secondary" @click="showAddCategoryForm = false; newCategory = { name: '', icon: '', is_default: false }">{{ t('settings.categories.cancel') }}</Button>
        </div>
      </div>

      <!-- Category List -->
      <div v-if="categories.length === 0 && !categoriesLoading" class="text-sm text-ink-muted italic p-4 bg-surface-2 rounded-lg text-center">
        {{ t('settings.categories.empty') }}
      </div>

      <div class="space-y-2">
        <!-- Default categories -->
        <div v-if="defaultCategories.length > 0">
          <div class="text-xs font-semibold text-ink-muted uppercase tracking-wide mb-2">
            {{ t('settings.categories.defaultSection') }}
          </div>
          <div
            v-for="cat in defaultCategories"
            :key="cat.id"
            class="border border-line rounded-xl overflow-hidden"
          >
            <div
              class="flex items-center gap-2 p-3 bg-surface cursor-pointer"
              @click="toggleCategory(cat.id)"
            >
              <span class="text-lg w-6 shrink-0 text-center">{{ cat.icon }}</span>
              <div class="flex-1 min-w-0">
                <span class="font-medium text-ink truncate block">{{ categoryLabel(cat) }}</span>
                <span class="text-xs text-ink-muted">{{ t('settings.categories.subcategoriesCount', { n: cat.subcategories?.length || 0 }) }}</span>
              </div>
              <div class="flex items-center shrink-0">
                <button
                  v-if="isAdmin"
                  @click.stop="startAddSubcategory(cat)"
                  class="p-1.5 text-blue-500 hover:text-blue-700 dark:hover:text-blue-400"
                  :aria-label="t('settings.categories.addSubcategoryAria')"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                  </svg>
                </button>
                <button
                  v-if="isAdmin"
                  @click.stop="deleteCategory(cat)"
                  class="p-1.5 text-red-400 hover:text-red-600 dark:hover:text-red-400"
                  :aria-label="t('settings.categories.deleteCategoryAria')"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
                <svg
                  :class="['w-4 h-4 ml-1 text-ink-faint transition-transform', expandedCategories.has(cat.id) ? 'rotate-180' : '']"
                  fill="none" stroke="currentColor" viewBox="0 0 24 24"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </div>

            <div v-if="expandedCategories.has(cat.id)" class="border-t border-line">
              <div v-if="addSubcategoryForCat === cat.id" class="p-3 bg-blue-50 dark:bg-blue-900/20 border-b border-blue-200 dark:border-blue-800">
                <div class="flex gap-2">
                  <input
                    v-model="newSubcategoryName"
                    type="text"
                    :placeholder="t('settings.categories.subcategoryPlaceholder')"
                    class="flex-1 px-2 py-2 text-base border border-line rounded
                           bg-surface text-ink
                           focus:outline-none focus:ring-1 focus:ring-blue-500"
                    @keyup.enter="saveSubcategory(cat.id)"
                    ref="subcategoryInput"
                  />
                  <button @click="saveSubcategory(cat.id)" class="px-3 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-700">{{ t('settings.categories.addSubcategoryButton') }}</button>
                  <button @click="addSubcategoryForCat = null; newSubcategoryName = ''" class="px-3 py-2 text-sm text-ink-soft hover:text-ink">{{ t('settings.categories.cancel') }}</button>
                </div>
              </div>

              <div v-if="!cat.subcategories?.length" class="px-4 py-2 text-sm text-ink-faint italic">
                {{ t('settings.categories.noSubcategories') }}
              </div>
              <div
                v-for="sub in cat.subcategories"
                :key="sub.id"
                class="flex items-center gap-2 px-4 py-2 hover:bg-surface-2"
              >
                <span class="w-4 h-4 text-ink-faint">·</span>
                <span class="flex-1 text-sm text-ink-soft">{{ categoryLabel(sub) }}</span>
                <button
                  v-if="isAdmin"
                  @click="deleteSubcategory(cat, sub)"
                  class="p-2 text-red-400 hover:text-red-600 opacity-60 hover:opacity-100"
                  :aria-label="t('settings.categories.deleteSubcategoryAria')"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Personal categories -->
        <div v-if="personalCategories.length > 0" :class="defaultCategories.length > 0 ? 'mt-4' : ''">
          <div class="text-xs font-semibold text-ink-muted uppercase tracking-wide mb-2">
            {{ t('settings.categories.personalSection') }}
          </div>
          <div
            v-for="cat in personalCategories"
            :key="cat.id"
            class="border border-line rounded-xl overflow-hidden"
          >
            <div
              class="flex items-center gap-2 p-3 bg-surface cursor-pointer"
              @click="toggleCategory(cat.id)"
            >
              <span class="text-lg w-6 shrink-0 text-center">{{ cat.icon }}</span>
              <div class="flex-1 min-w-0">
                <span class="font-medium text-ink truncate block">{{ categoryLabel(cat) }}</span>
                <span class="text-xs text-ink-muted">{{ t('settings.categories.subcategoriesCount', { n: cat.subcategories?.length || 0 }) }}</span>
              </div>
              <div class="flex items-center shrink-0">
                <button
                  @click.stop="startAddSubcategory(cat)"
                  class="p-1.5 text-blue-500 hover:text-blue-700 dark:hover:text-blue-400"
                  :aria-label="t('settings.categories.addSubcategoryAria')"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                  </svg>
                </button>
                <button
                  @click.stop="deleteCategory(cat)"
                  class="p-1.5 text-red-400 hover:text-red-600 dark:hover:text-red-400"
                  :aria-label="t('settings.categories.deleteCategoryAria')"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
                <svg
                  :class="['w-4 h-4 ml-1 text-ink-faint transition-transform', expandedCategories.has(cat.id) ? 'rotate-180' : '']"
                  fill="none" stroke="currentColor" viewBox="0 0 24 24"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </div>

            <div v-if="expandedCategories.has(cat.id)" class="border-t border-line">
              <div v-if="addSubcategoryForCat === cat.id" class="p-3 bg-blue-50 dark:bg-blue-900/20 border-b border-blue-200 dark:border-blue-800">
                <div class="flex gap-2">
                  <input
                    v-model="newSubcategoryName"
                    type="text"
                    :placeholder="t('settings.categories.subcategoryPlaceholder')"
                    class="flex-1 px-2 py-2 text-base border border-line rounded
                           bg-surface text-ink
                           focus:outline-none focus:ring-1 focus:ring-blue-500"
                    @keyup.enter="saveSubcategory(cat.id)"
                  />
                  <button @click="saveSubcategory(cat.id)" class="px-3 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-700">{{ t('settings.categories.addSubcategoryButton') }}</button>
                  <button @click="addSubcategoryForCat = null; newSubcategoryName = ''" class="px-3 py-2 text-sm text-ink-soft hover:text-ink">{{ t('settings.categories.cancel') }}</button>
                </div>
              </div>

              <div v-if="!cat.subcategories?.length" class="px-4 py-2 text-sm text-ink-faint italic">
                {{ t('settings.categories.noSubcategories') }}
              </div>
              <div
                v-for="sub in cat.subcategories"
                :key="sub.id"
                class="flex items-center gap-2 px-4 py-2 hover:bg-surface-2"
              >
                <span class="w-4 h-4 text-ink-faint">·</span>
                <span class="flex-1 text-sm text-ink-soft">{{ categoryLabel(sub) }}</span>
                <button
                  @click="deleteSubcategory(cat, sub)"
                  class="p-2 text-red-400 hover:text-red-600 opacity-60 hover:opacity-100"
                  :aria-label="t('settings.categories.deleteSubcategoryAria')"
                >
                  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="categoryError" class="mt-3 text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
        {{ categoryError }}
      </div>
    </Card>
  </div>
</template>

<script setup>
defineOptions({ name: 'CategoriesTab' })

import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import { categoriesAPI } from '@/api/client'
import { categoryLabel } from '@/utils/categoryLabel'
import Card from '@/components/common/Card.vue'
import Button from '@/components/common/Button.vue'
import { apiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const { confirm } = useConfirm()

const isAdmin = computed(() => settingsStore.isPropertyAdmin)

const categories = ref([])
const categoriesLoading = ref(false)
const categoryError = ref(null)
const expandedCategories = ref(new Set())
const showAddCategoryForm = ref(false)
const newCategory = ref({ name: '', icon: '', is_default: false })
const addSubcategoryForCat = ref(null)
const newSubcategoryName = ref('')

const defaultCategories = computed(() => categories.value.filter(c => c.is_default))
const personalCategories = computed(() => categories.value.filter(c => !c.is_default))

function toggleCategory(id) {
  if (expandedCategories.value.has(id)) {
    expandedCategories.value.delete(id)
  } else {
    expandedCategories.value.add(id)
  }
  expandedCategories.value = new Set(expandedCategories.value)
}

async function fetchCategories() {
  categoriesLoading.value = true
  try {
    const { data } = await categoriesAPI.list()
    categories.value = data || []
  } catch (err) {
    console.error('Error fetching categories:', err)
  } finally {
    categoriesLoading.value = false
  }
}

async function addCategory() {
  if (!newCategory.value.name.trim()) return
  categoryError.value = null
  try {
    await categoriesAPI.create({
      name: newCategory.value.name.trim(),
      icon: newCategory.value.icon.trim() || '📁',
      is_default: isAdmin.value && newCategory.value.is_default
    })
    newCategory.value = { name: '', icon: '', is_default: false }
    showAddCategoryForm.value = false
    await fetchCategories()
  } catch (err) {
    categoryError.value = apiErrorMessage(err, t('settings.categories.createError'))
  }
}

async function deleteCategory(cat) {
  const ok = await confirm({
    title: t('settings.categories.deleteCategoryTitle'),
    message: cat.is_default
      ? t('settings.categories.deleteCategoryMessageDefault', { name: categoryLabel(cat) })
      : t('settings.categories.deleteCategoryMessagePersonal', { name: categoryLabel(cat) }),
    confirmText: t('settings.categories.deleteCategoryConfirm'),
    variant: 'danger'
  })
  if (!ok) return
  categoryError.value = null
  try {
    await categoriesAPI.delete(cat.id)
    await fetchCategories()
  } catch (err) {
    categoryError.value = apiErrorMessage(err, t('settings.categories.deleteError'))
  }
}

function startAddSubcategory(cat) {
  addSubcategoryForCat.value = cat.id
  newSubcategoryName.value = ''
  if (!expandedCategories.value.has(cat.id)) {
    expandedCategories.value.add(cat.id)
    expandedCategories.value = new Set(expandedCategories.value)
  }
}

async function saveSubcategory(catId) {
  if (!newSubcategoryName.value.trim()) return
  categoryError.value = null
  try {
    await categoriesAPI.createSubcategory(catId, { name: newSubcategoryName.value.trim() })
    addSubcategoryForCat.value = null
    newSubcategoryName.value = ''
    await fetchCategories()
  } catch (err) {
    categoryError.value = apiErrorMessage(err, t('settings.categories.createError'))
  }
}

async function deleteSubcategory(cat, sub) {
  const ok = await confirm({
    title: t('settings.categories.deleteSubcategoryTitle'),
    message: t('settings.categories.deleteSubcategoryMessage', { name: categoryLabel(sub) }),
    confirmText: t('settings.categories.deleteCategoryConfirm'),
    variant: 'danger'
  })
  if (!ok) return
  categoryError.value = null
  try {
    await categoriesAPI.deleteSubcategory(cat.id, sub.id)
    await fetchCategories()
  } catch (err) {
    categoryError.value = apiErrorMessage(err, t('settings.categories.deleteError'))
  }
}

onMounted(() => {
  fetchCategories()
})
</script>
